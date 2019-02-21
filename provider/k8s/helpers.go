package k8s

import (
	"archive/tar"
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
	cc "github.com/convox/rack/provider/k8s/pkg/client/clientset/versioned/typed/convox/v1"
	ac "k8s.io/api/core/v1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ScannerStartSize = 4096
	ScannerMaxSize   = 1024 * 1024
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
	pidch <- ps.Id
}

func (p *Provider) systemEnvironment(app, release string) (map[string]string, error) {
	senv := map[string]string{
		"APP":      app,
		"RACK":     p.Rack,
		"RACK_URL": fmt.Sprintf("https://convox:%s@api.%s.svc.cluster.local:5443", p.Password, p.Rack),
		"RELEASE":  release,
	}

	if cs, _ := p.Cluster.CoreV1().Secrets("convox-system").Get("ca", am.GetOptions{}); cs != nil {
		if ca := cs.Data["tls.crt"]; ca != nil {
			senv["RACK_CA"] = base64.StdEncoding.EncodeToString(ca)
		}
	}

	r, err := p.ReleaseGet(app, release)
	if err != nil {
		return nil, err
	}

	if r.Build != "" {
		b, err := p.BuildGet(app, r.Build)
		if err != nil {
			return nil, err
		}

		senv["BUILD"] = b.Id
		senv["BUILD_DESCRIPTION"] = b.Description
	}

	return senv, nil
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

func envName(s string) string {
	return strings.Replace(strings.ToUpper(s), "-", "_", -1)
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

func streamLogsWithPrefix(w io.WriteCloser, r io.Reader, prefix string) {
	defer w.Close()

	ls := bufio.NewScanner(r)

	ls.Buffer(make([]byte, ScannerStartSize), ScannerMaxSize)

	for ls.Scan() {
		parts := strings.SplitN(ls.Text(), " ", 2)

		ts, err := time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			fmt.Printf("err = %+v\n", err)
			continue
		}

		fmt.Fprintf(w, "%s %s %s\n", ts.Format(helpers.PrintableTime), prefix, parts[1])
	}

	if err := ls.Err(); err != nil {
		fmt.Fprintf(w, "%s %s scan error: %s\n", time.Now().Format(helpers.PrintableTime), prefix, err)
	}
}

func systemVolume(v string) bool {
	switch v {
	case "/var/run/docker.sock":
		return true
	case "/var/snap/microk8s/current/docker.sock":
		return true
	}
	return false
}

func (p *Provider) volumeFrom(app, service, v string) string {
	if from := strings.Split(v, ":")[0]; systemVolume(from) {
		return from
	} else if strings.Contains(v, ":") {
		return path.Join("/mnt/volumes", app, "app", from)
	} else {
		return path.Join("/mnt/volumes", app, "service", service, from)
	}
}

func (p *Provider) volumeName(app, v string) string {
	hash := sha256.Sum256([]byte(v))
	return fmt.Sprintf("%s-%s-%x", p.Rack, app, hash[0:20])
}

func (p *Provider) volumeSources(app, service string, vs []string) []string {
	vsh := map[string]bool{}

	for _, v := range vs {
		vsh[p.volumeFrom(app, service, v)] = true
	}

	vsu := []string{}

	for v := range vsh {
		vsu = append(vsu, v)
	}

	sort.Strings(vsu)

	return vsu
}

func volumeTo(v string) (string, error) {
	switch parts := strings.SplitN(v, ":", 2); len(parts) {
	case 1:
		return parts[0], nil
	case 2:
		return parts[1], nil
	default:
		return "", fmt.Errorf("invalid volume %q", v)
	}
}
