package k8s

import (
	"io"
	"io/ioutil"

	ac "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

func (p *Provider) FilesDelete(app, pid string, files []string) error {
	req := p.Cluster.CoreV1().RESTClient().Post().Resource("pods").Name(pid).Namespace(p.appNamespace(app)).SubResource("exec").Param("container", "main")

	command := []string{"rm", "-f"}
	command = append(command, files...)

	eo := &ac.PodExecOptions{
		Container: "main",
		Command:   command,
		Stdout:    true,
	}

	req.VersionedParams(eo, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(p.Config, "POST", req.URL())
	if err != nil {
		return err
	}

	if err := exec.Stream(remotecommand.StreamOptions{Stdout: ioutil.Discard}); err != nil {
		return err
	}

	return nil
}

func (p *Provider) FilesDownload(app, pid, file string) (io.Reader, error) {
	req := p.Cluster.CoreV1().RESTClient().Post().Resource("pods").Name(pid).Namespace(p.appNamespace(app)).SubResource("exec").Param("container", "main")

	eo := &ac.PodExecOptions{
		Container: "main",
		Command:   []string{"tar", "czPf", "-", file},
		Stdout:    true,
	}

	req.VersionedParams(eo, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(p.Config, "POST", req.URL())
	if err != nil {
		return nil, err
	}

	r, w := io.Pipe()

	go func() {
		exec.Stream(remotecommand.StreamOptions{Stdout: w})
		w.Close()
	}()

	return r, nil
}

func (p *Provider) FilesUpload(app, pid string, r io.Reader) error {
	req := p.Cluster.CoreV1().RESTClient().Post().Resource("pods").Name(pid).Namespace(p.appNamespace(app)).SubResource("exec").Param("container", "main")

	eo := &ac.PodExecOptions{
		Container: "main",
		Command:   []string{"tar", "xzPf", "-"},
		Stdin:     true,
	}

	req.VersionedParams(eo, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(p.Config, "POST", req.URL())
	if err != nil {
		return err
	}

	if err := exec.Stream(remotecommand.StreamOptions{Stdin: r}); err != nil {
		return err
	}

	return nil
}
