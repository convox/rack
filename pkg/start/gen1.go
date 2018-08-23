package start

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/manifest1"
)

func Start1(opts Options) error {
	opts.Manifest = helpers.Coalesce(opts.Manifest, "docker-compose.yml")

	if !helpers.FileExists(opts.Manifest) {
		return fmt.Errorf("manifest not found: %s", opts.Manifest)
	}

	m, err := manifest1.LoadFile(opts.Manifest)
	if err != nil {
		return err
	}

	errs := m.Validate()

	switch len(errs) {
	case 0:
	case 1:
		return errs[0]
	default:
		ss := []string{""}
		for _, err := range errs {
			ss = append(ss, err.Error())
		}
		return errors.New(strings.Join(ss, "\n"))
	}

	service := ""

	if opts.Services != nil {
		if len(opts.Services) > 1 {
			return fmt.Errorf("can not specify multiple services for gen1 apps")
		}

		service = opts.Services[0]

		if _, ok := m.Services[service]; !ok {
			return fmt.Errorf("service not found in manifest: %s", service)
		}
	}

	if err := m.Shift(opts.Shift); err != nil {
		return err
	}

	pcc, err := m.PortConflicts()
	if err != nil {
		return err
	}
	if len(pcc) > 0 {
		return fmt.Errorf("ports in use: %v", pcc)
	}

	r := m.Run(filepath.Dir(opts.Manifest), opts.App, manifest1.RunOptions{
		Build:   opts.Build,
		Cache:   opts.Cache,
		Command: opts.Command,
		Service: service,
		Sync:    opts.Sync,
	})

	err = r.Start()
	if err != nil {
		r.Stop()
		return err
	}

	go handleInterrupt(func() { r.Stop() })

	return r.Wait()
}
