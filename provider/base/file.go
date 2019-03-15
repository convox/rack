package base

import (
	"fmt"
	"io"
)

func (p *Provider) FilesDelete(app, pid string, files []string) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) FilesDownload(app, pid, file string) (io.Reader, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) FilesUpload(app, pid string, r io.Reader) error {
	return fmt.Errorf("unimplemented")
}
