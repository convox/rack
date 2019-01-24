package k8s

import (
	"crypto/sha256"
	"fmt"
	"io"
	"strings"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	shellquote "github.com/kballard/go-shellquote"
	ac "k8s.io/api/core/v1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/exec"
)

func (p *Provider) ProcessExec(app, pid, command string, rw io.ReadWriter, opts structs.ProcessExecOptions) (int, error) {
	req := p.Cluster.CoreV1().RESTClient().Post().Resource("pods").Name(pid).Namespace(p.AppNamespace(app)).SubResource("exec").Param("container", "main")

	cp, err := shellquote.Split(command)
	if err != nil {
		return 0, err
	}

	eo := &ac.PodExecOptions{
		Container: "main",
		Command:   cp,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}

	req.VersionedParams(eo, scheme.ParameterCodec)

	e, err := remotecommand.NewSPDYExecutor(p.Config, "POST", req.URL())
	if err != nil {
		return 0, err
	}

	err = e.Stream(remotecommand.StreamOptions{Stdin: rw, Stdout: rw, Stderr: rw, Tty: true})
	if ee, ok := err.(exec.ExitError); ok {
		return ee.ExitStatus(), nil
	}
	if err != nil {
		return 0, err
	}

	return 0, nil
}

func (p *Provider) ProcessGet(app, pid string) (*structs.Process, error) {
	pd, err := p.Cluster.CoreV1().Pods(p.AppNamespace(app)).Get(pid, am.GetOptions{})
	if err != nil {
		return nil, err
	}

	ps, err := processFromPod(*pd)
	if err != nil {
		return nil, err
	}

	return ps, nil
}

func (p *Provider) ProcessList(app string, opts structs.ProcessListOptions) (structs.Processes, error) {
	filters := []string{
		"type!=resource",
	}

	if opts.Release != nil {
		filters = append(filters, fmt.Sprintf("release=%s", *opts.Release))
	}

	if opts.Service != nil {
		filters = append(filters, fmt.Sprintf("service=%s", *opts.Service))
	}

	pds, err := p.Cluster.CoreV1().Pods(p.AppNamespace(app)).List(am.ListOptions{LabelSelector: strings.Join(filters, ",")})
	if err != nil {
		return nil, err
	}

	pss := structs.Processes{}

	for _, pd := range pds.Items {
		ps, err := processFromPod(pd)
		if err != nil {
			return nil, err
		}

		pss = append(pss, *ps)
	}

	return pss, nil
}

func (p *Provider) ProcessLogs(app, pid string, opts structs.LogsOptions) (io.ReadCloser, error) {
	ps, err := p.ProcessGet(app, pid)
	if err != nil {
		return nil, err
	}

	r, w := io.Pipe()

	go p.streamProcessLogs(w, *ps, opts)

	return r, nil
}

func (p *Provider) ProcessRun(app, service string, opts structs.ProcessRunOptions) (*structs.Process, error) {
	s, err := p.podSpecFromRunOptions(app, service, opts)
	if err != nil {
		return nil, err
	}

	ns, err := p.Cluster.CoreV1().Namespaces().Get(p.Rack, am.GetOptions{})
	if err != nil {
		return nil, err
	}

	pd, err := p.Cluster.CoreV1().Pods(p.AppNamespace(app)).Create(&ac.Pod{
		ObjectMeta: am.ObjectMeta{
			Annotations: map[string]string{
				"iam.amazonaws.com/role": ns.ObjectMeta.Annotations["convox.aws.role"],
			},
			GenerateName: fmt.Sprintf("%s-", service),
			Labels: map[string]string{
				"system":  "convox",
				"rack":    p.Rack,
				"app":     app,
				"service": service,
				"type":    "process",
			},
		},
		Spec: *s,
	})
	if err != nil {
		return nil, err
	}

	ps, err := p.ProcessGet(app, pd.ObjectMeta.Name)
	if err != nil {
		return nil, err
	}

	return ps, nil
}

func (p *Provider) ProcessStop(app, pid string) error {
	if err := p.Cluster.CoreV1().Pods(p.AppNamespace(app)).Delete(pid, nil); err != nil {
		return err
	}

	return nil
}

func (p *Provider) ProcessWait(app, pid string) (int, error) {
	for {
		pd, err := p.Cluster.CoreV1().Pods(p.AppNamespace(app)).Get(pid, am.GetOptions{})
		if err != nil {
			return 0, err
		}

		cs := pd.Status.ContainerStatuses

		if len(cs) != 1 || cs[0].Name != "main" {
			return 0, fmt.Errorf("unexpected containers for pid: %s", pid)
		}

		if t := cs[0].State.Terminated; t != nil {
			if err := p.ProcessStop(app, pid); err != nil {
				return 0, err
			}

			return int(t.ExitCode), nil
		}
	}
}

func (p *Provider) podSpecFromService(app, service, release string) (*ac.PodSpec, error) {
	if release == "" {
		a, err := p.AppGet(app)
		if err != nil {
			return nil, err
		}

		release = a.Release
	}

	c := ac.Container{
		Env:           []ac.EnvVar{},
		Name:          "main",
		VolumeDevices: []ac.VolumeDevice{},
		VolumeMounts:  []ac.VolumeMount{},
	}

	vs := []ac.Volume{}

	if release != "" {
		m, r, err := helpers.ReleaseManifest(p, app, release)
		if err != nil {
			return nil, err
		}

		env := map[string]string{}

		senv, err := p.systemEnvironment(app, release)
		if err != nil {
			return nil, err
		}

		for k, v := range senv {
			env[k] = v
		}

		e := structs.Environment{}

		if err := e.Load([]byte(r.Env)); err != nil {
			return nil, err
		}

		for k, v := range e {
			env[k] = v
		}

		if s, _ := m.Service(service); s != nil {
			if s.Command != "" {
				parts, err := shellquote.Split(s.Command)
				if err != nil {
					return nil, err
				}
				c.Args = parts
			}

			for k, v := range s.EnvironmentDefaults() {
				env[k] = v
			}

			for _, l := range s.Links {
				env[fmt.Sprintf("%s_URL", envName(l))] = fmt.Sprintf("https://%s.%s.%s", l, app, p.Rack)
			}

			for _, r := range s.Resources {
				cm, err := p.Cluster.CoreV1().ConfigMaps(p.AppNamespace(app)).Get(fmt.Sprintf("resource-%s", r), am.GetOptions{})
				if err != nil {
					return nil, err
				}

				env[fmt.Sprintf("%s_URL", envName(r))] = cm.Data["URL"]
			}

			repo, _, err := p.Engine.AppRepository(app)
			if err != nil {
				return nil, err
			}

			c.Image = fmt.Sprintf("%s:%s.%s", repo, service, r.Build)

			for _, v := range s.Volumes {
				vv, vm := podVolume(app, v, v)
				vs = append(vs, vv)
				c.VolumeMounts = append(c.VolumeMounts, vm)
			}
		}

		for k, v := range env {
			c.Env = append(c.Env, ac.EnvVar{Name: k, Value: v})
		}
	}

	ps := &ac.PodSpec{
		Containers:            []ac.Container{c},
		ShareProcessNamespace: options.Bool(true),
		Volumes:               vs,
	}

	if ip, err := p.Engine.Resolver(); err == nil {
		ps.DNSPolicy = "None"
		ps.DNSConfig = &ac.PodDNSConfig{
			Nameservers: []string{ip},
		}
	}

	return ps, nil
}

func (p *Provider) podSpecFromRunOptions(app, service string, opts structs.ProcessRunOptions) (*ac.PodSpec, error) {
	s, err := p.podSpecFromService(app, service, helpers.DefaultString(opts.Release, ""))
	if err != nil {
		return nil, err
	}

	if opts.Command != nil {
		parts, err := shellquote.Split(*opts.Command)
		if err != nil {
			return nil, err
		}
		s.Containers[0].Args = parts
	}

	if opts.Environment != nil {
		for k, v := range opts.Environment {
			s.Containers[0].Env = append(s.Containers[0].Env, ac.EnvVar{Name: k, Value: v})
		}
	}

	if opts.Image != nil {
		s.Containers[0].Image = *opts.Image
	}

	if opts.Volumes != nil {
		for from, to := range opts.Volumes {
			v, vm := podVolume(app, from, to)
			s.Volumes = append(s.Volumes, v)
			s.Containers[0].VolumeMounts = append(s.Containers[0].VolumeMounts, vm)
		}
	}

	s.RestartPolicy = "Never"

	return s, nil
}

func podVolume(app, from, to string) (ac.Volume, ac.VolumeMount) {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%s", from, to)))
	name := fmt.Sprintf("volume-%x", hash[0:20])

	host := &ac.HostPathVolumeSource{
		Path: from,
	}

	if !systemVolume(from) {
		t := ac.HostPathDirectoryOrCreate
		host.Path = fmt.Sprintf("/mnt/volumes/%s/%s", app, from)
		host.Type = &t
	}

	v := ac.Volume{
		Name:         name,
		VolumeSource: ac.VolumeSource{HostPath: host},
	}

	vm := ac.VolumeMount{
		Name:      name,
		MountPath: to,
	}

	return v, vm

}

func processFromPod(pd ac.Pod) (*structs.Process, error) {
	cs := pd.Spec.Containers

	if len(cs) != 1 || cs[0].Name != "main" {
		return nil, fmt.Errorf("unexpected containers for pid: %s", pd.ObjectMeta.Name)
	}

	status := "unknown"

	switch pd.Status.Phase {
	case "Failed":
		status = "failed"
	case "Pending":
		status = "pending"
	case "Running":
		status = "running"
	case "Succeeded":
		status = "complete"
	}

	ps := &structs.Process{
		Id:       pd.ObjectMeta.Name,
		App:      pd.ObjectMeta.Labels["app"],
		Command:  shellquote.Join(cs[0].Args...),
		Host:     "",
		Image:    cs[0].Image,
		Instance: "",
		Name:     pd.ObjectMeta.Labels["service"],
		Release:  pd.ObjectMeta.Labels["release"],
		Started:  pd.CreationTimestamp.Time,
		Status:   status,
	}

	return ps, nil
}
