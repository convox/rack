package aws

import (
	"fmt"
	"io"

	docker "github.com/fsouza/go-dockerclient"
)

func (p *Provider) FilesDelete(app, pid string, files []string) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) FilesDownload(app, pid string, file string) (io.Reader, error) {
	log := p.logger("FilesDownload").Append("app=%q pid=%q file=%q", app, pid, file)

	if _, err := p.AppGet(app); err != nil {
		return nil, log.Error(err)
	}

	dc, err := p.dockerClientFromPid(pid)
	if err != nil {
		return nil, log.Error(err)
	}

	c, err := p.dockerContainerFromPid(pid)
	if err != nil {
		return nil, log.Error(err)
	}

	r, w := io.Pipe()

	opts := docker.DownloadFromContainerOptions{
		OutputStream: w,
		Path:         file,
	}

	go func() {
		err := dc.DownloadFromContainer(c.ID, opts)
		if err != nil {
			log.Error(err)
		}

		w.Close()
	}()

	return r, log.Success()
}

func (p *Provider) FilesUpload(app, pid string, r io.Reader) error {
	log := p.logger("FilesUpload").Append("app=%q pid=%q", app, pid)

	if _, err := p.AppGet(app); err != nil {
		return log.Error(err)
	}

	dc, err := p.dockerClientFromPid(pid)
	if err != nil {
		return log.Error(err)
	}

	c, err := p.dockerContainerFromPid(pid)
	if err != nil {
		return log.Error(err)
	}

	opts := docker.UploadToContainerOptions{
		InputStream: r,
		Path:        "/",
	}

	if err := dc.UploadToContainer(c.ID, opts); err != nil {
		return log.Error(err)
	}

	return log.Success()
}
