package k8s

import (
	"archive/tar"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
	cc "github.com/convox/rack/provider/k8s/pkg/client/clientset/versioned/typed/convox/v1"
	ac "k8s.io/api/core/v1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type processLister func() (structs.Processes, error)

func (p *Provider) convoxClient() (cc.ConvoxV1Interface, error) {
	return cc.NewForConfig(p.Config)
}

func (p *Provider) podLogs(namespace, name string, opts structs.LogsOptions) (io.ReadCloser, error) {
	if err := p.podWaitForRunning(namespace, name); err != nil {
		return nil, err
	}

	lopts := &ac.PodLogOptions{
		// Container:  "main",
		Follow:     helpers.DefaultBool(opts.Follow, true),
		Timestamps: helpers.DefaultBool(opts.Prefix, false),
	}

	if opts.Since != nil {
		t := am.NewTime(time.Now().UTC().Add(-1 * *opts.Since))
		lopts.SinceTime = &t
	}

	r, err := p.Cluster.CoreV1().Pods(namespace).GetLogs(name, lopts).Stream()
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (p *Provider) podWaitForRunning(namespace, name string) error {
	for {
		pd, err := p.Cluster.CoreV1().Pods(namespace).Get(name, am.GetOptions{})
		if err != nil {
			return err
		}

		for _, c := range pd.Status.ContainerStatuses {
			if c.State.Waiting == nil {
				return nil
			}
		}

		time.Sleep(1 * time.Second)
	}
}

func (p *Provider) streamProcessListLogs(w io.WriteCloser, pl processLister, opts structs.LogsOptions, ch chan error) {
	defer w.Close()
	defer close(ch)

	current := map[string]bool{}

	follow := helpers.DefaultBool(opts.Follow, false)
	pidch := make(chan string)
	tick := time.NewTicker(1 * time.Second)

	for {
		if _, err := w.Write([]byte{}); err != nil {
			ch <- err
			return
		}

		select {
		case pid := <-pidch:
			delete(current, pid)

			if len(current) == 0 && !follow {
				tick.Stop()
				return
			}
		case <-tick.C:
			pss, err := pl()
			if err != nil {
				ch <- err
				continue
			}

			for _, ps := range pss {
				if !current[ps.Id] {
					current[ps.Id] = true
					go p.streamProcessLogsWait(w, ps, opts, pidch)
				}
			}
		}
	}
}

func (p *Provider) streamProcessLogsWait(w io.WriteCloser, ps structs.Process, opts structs.LogsOptions, pidch chan string) {
	ri, wi := io.Pipe()
	go p.streamProcessLogs(wi, ps, opts)
	io.Copy(w, ri)
	pps, err := p.ProcessGet(ps.App, ps.Id)
	if err != nil {
		pidch <- ps.Id
	}
	if pps != nil && pps.Status == "running" {
		pidch <- ps.Id
	}
}

func (p *Provider) streamProcessLogs(w io.WriteCloser, ps structs.Process, opts structs.LogsOptions) {
	defer w.Close()

	r, err := p.podLogs(p.AppNamespace(ps.App), ps.Id, opts)
	if err != nil {
		return
	}
	defer r.Close()

	if helpers.DefaultBool(opts.Prefix, false) {
		streamLogsWithPrefix(w, r, fmt.Sprintf("service/%s:%s/%s", ps.Name, ps.Release, ps.Id))
	} else {
		io.Copy(w, r)
	}
}

func dockerSystemId() (string, error) {
	data, err := exec.Command("docker", "system", "info").CombinedOutput()
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "ID: ") {
			return strings.ToLower(strings.TrimPrefix(line, "ID: ")), nil
		}
	}

	return "", fmt.Errorf("could not find docker system id")
}

func streamLogsWithPrefix(w io.WriteCloser, r io.Reader, prefix string) {
	defer w.Close()

	ls := bufio.NewScanner(r)

	for ls.Scan() {
		parts := strings.SplitN(ls.Text(), " ", 2)

		ts, err := time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			fmt.Printf("err = %+v\n", err)
			continue
		}

		fmt.Fprintf(w, "%s %s %s\n", ts.Format(helpers.PrintableTime), prefix, parts[1])
	}
}

type imageManifest []struct {
	RepoTags []string
}

func extractImageManifest(r io.Reader) (imageManifest, error) {
	mtr := tar.NewReader(r)

	var manifest imageManifest

	for {
		mh, err := mtr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if mh.Name == "manifest.json" {
			var mdata bytes.Buffer

			if _, err := io.Copy(&mdata, mtr); err != nil {
				return nil, err
			}

			if err := json.Unmarshal(mdata.Bytes(), &manifest); err != nil {
				return nil, err
			}

			return manifest, nil
		}
	}

	return nil, fmt.Errorf("unable to locate manifest")
}

func processFilter(in structs.Processes, fn func(structs.Process) bool) structs.Processes {
	out := structs.Processes{}

	for _, ps := range in {
		if fn(ps) {
			out = append(out, ps)
		}
	}

	return out
}

func systemVolume(v string) bool {
	switch v {
	case "/var/run/docker.sock":
		return true
	}
	return false
}
