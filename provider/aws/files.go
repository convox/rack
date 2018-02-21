package aws

import (
	"fmt"
	"io"
)

func (p *AWSProvider) FilesDelete(app, pid string, files []string) error {
	return fmt.Errorf("unimplemented")
}

func (p *AWSProvider) FilesUpload(app, pid string, r io.Reader) error {
	return fmt.Errorf("unimplemented")
}
