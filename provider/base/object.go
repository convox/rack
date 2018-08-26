package base

import (
	"fmt"
	"io"

	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) ObjectDelete(app, key string) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) ObjectExists(app, key string) (bool, error) {
	return false, fmt.Errorf("unimplemented")
}

func (p *Provider) ObjectFetch(app, key string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) ObjectList(app, prefix string) ([]string, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) ObjectStore(app, key string, r io.Reader, opts structs.ObjectStoreOptions) (*structs.Object, error) {
	return nil, fmt.Errorf("unimplemented")
}
