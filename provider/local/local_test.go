package local_test

import (
	"io/ioutil"
	"os"

	"github.com/convox/rack/provider/local"
	"github.com/convox/rack/structs"
)

func testProvider() (*local.Provider, error) {
	tmp, err := ioutil.TempDir("", "rack")
	if err != nil {
		return nil, err
	}

	p := &local.Provider{
		Root:   tmp,
		Router: "none",
		Test:   true,
	}

	if err := p.Initialize(structs.ProviderOptions{}); err != nil {
		return nil, err
	}

	return p, nil
}

func testProviderCleanup(p *local.Provider) {
	if p.Root != "" {
		os.RemoveAll(p.Root)
	}
}
