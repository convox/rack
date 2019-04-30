package helpers

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/pkg/errors"
)

var (
	ProviderWaitDuration = 5 * time.Second
)

func AppEnvironment(p structs.Provider, app string) (structs.Environment, error) {
	rs, err := ReleaseLatest(p, app)
	if err != nil {
		return nil, err
	}
	if rs == nil {
		return structs.Environment{}, nil
	}

	env := structs.Environment{}

	if err := env.Load([]byte(rs.Env)); err != nil {
		return nil, err
	}

	return env, nil
}

func AppManifest(p structs.Provider, app string) (*manifest.Manifest, *structs.Release, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, nil, err
	}

	if a.Release == "" {
		return nil, nil, errors.WithStack(fmt.Errorf("no release for app: %s", app))
	}

	return ReleaseManifest(p, app, a.Release)
}

func ReleaseLatest(p structs.Provider, app string) (*structs.Release, error) {
	rs, err := p.ReleaseList(app, structs.ReleaseListOptions{Limit: options.Int(1)})
	if err != nil {
		return nil, err
	}

	if len(rs) < 1 {
		return nil, nil
	}

	return p.ReleaseGet(app, rs[0].Id)
}

func ReleaseManifest(p structs.Provider, app, release string) (*manifest.Manifest, *structs.Release, error) {
	r, err := p.ReleaseGet(app, release)
	if err != nil {
		return nil, nil, err
	}

	env := structs.Environment{}

	if err := env.Load([]byte(r.Env)); err != nil {
		return nil, nil, err
	}

	m, err := manifest.Load([]byte(r.Manifest), env)
	if err != nil {
		return nil, nil, err
	}

	return m, r, nil
}

func StreamAppLogs(ctx context.Context, p structs.Provider, w io.Writer, app string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		r, err := p.AppLogs(app, structs.LogsOptions{Prefix: options.Bool(true), Since: options.Duration(5 * time.Second)})
		if err != nil {
			return
		}

		copySystemLogs(ctx, w, r)

		time.Sleep(1 * time.Second)
	}
}

func StreamSystemLogs(ctx context.Context, p structs.Provider, w io.Writer) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		r, err := p.SystemLogs(structs.LogsOptions{Prefix: options.Bool(true), Since: options.Duration(5 * time.Second)})
		if err != nil {
			return
		}

		copySystemLogs(ctx, w, r)

		time.Sleep(1 * time.Second)
	}
}

func WaitForAppDeleted(p structs.Provider, w io.Writer, app string) error {
	time.Sleep(ProviderWaitDuration) // give the stack time to start updating

	return Wait(ProviderWaitDuration, 35*time.Minute, 2, func() (bool, error) {
		_, err := p.AppGet(app)
		if err == nil {
			return false, nil
		}
		if strings.Contains(err.Error(), "no such app") {
			return true, nil
		}
		if strings.Contains(err.Error(), "app not found") {
			return true, nil
		}
		return false, err
	})
}

func WaitForAppRunning(p structs.Provider, app string) error {
	return WaitForAppRunningContext(context.Background(), p, app)
}

func WaitForAppRunningContext(ctx context.Context, p structs.Provider, app string) error {
	time.Sleep(ProviderWaitDuration) // give the stack time to start updating

	var waitError error

	return WaitContext(ctx, ProviderWaitDuration, 35*time.Minute, 2, func() (bool, error) {
		a, err := p.AppGet(app)
		if err != nil {
			return false, err
		}

		if a.Status == "rollback" {
			waitError = fmt.Errorf("rollback")
		}

		return a.Status == "running", waitError
	})
}

func WaitForAppWithLogs(p structs.Provider, w io.Writer, app string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	return WaitForAppWithLogsContext(ctx, p, w, app)
}

func WaitForAppWithLogsContext(ctx context.Context, p structs.Provider, w io.Writer, app string) error {
	go StreamAppLogs(ctx, p, w, app)

	if err := WaitForAppRunningContext(ctx, p, app); err != nil {
		return err
	}

	return nil
}

func WaitForProcessRunning(p structs.Provider, w io.Writer, app, pid string) error {
	return Wait(1*time.Second, 5*time.Minute, 2, func() (bool, error) {
		ps, err := p.ProcessGet(app, pid)
		if err != nil {
			return false, err
		}

		return ps.Status == "running", nil
	})
}

func WaitForRackRunning(p structs.Provider, w io.Writer) error {
	time.Sleep(ProviderWaitDuration) // give the stack time to start updating

	return Wait(ProviderWaitDuration, 35*time.Minute, 2, func() (bool, error) {
		s, err := p.SystemGet()
		if err != nil {
			return false, err
		}

		return s.Status == "running", nil
	})
}

func WaitForRackWithLogs(p structs.Provider, w io.Writer) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go StreamSystemLogs(ctx, p, w)

	if err := WaitForRackRunning(p, w); err != nil {
		return err
	}

	return nil
}

func copySystemLogs(ctx context.Context, w io.Writer, r io.Reader) {
	s := bufio.NewScanner(r)

	for s.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		parts := strings.SplitN(s.Text(), " ", 3)

		if len(parts) < 3 {
			continue
		}

		if strings.HasPrefix(parts[1], "system/") {
			w.Write([]byte(fmt.Sprintf("%s\n", s.Text())))
		}
	}
}
