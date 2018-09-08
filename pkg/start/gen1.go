package start

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/manifest1"
)

type Options1 struct {
	App      string
	Build    bool
	Cache    bool
	Command  []string
	Manifest string
	Service  string
	Shift    int
	Sync     bool
}

func (s *Start) Start1(ctx context.Context, opts Options1) error {
	select {
	case <-ctx.Done():
		return nil
	default:
	}
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
		Service: opts.Service,
		Sync:    opts.Sync,
	})

	defer r.Stop()

	if err := r.Start(); err != nil {
		return err
	}

	return r.Wait(ctx)
}
