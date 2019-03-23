package local

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/logstore"
	"github.com/convox/rack/pkg/structs"
)

var logs = logstore.New()

func (p *Provider) Log(group, stream string, ts time.Time, message string) error {
	logs.Append(group, ts, stream, message)
	logs.Append(fmt.Sprintf("%s/%s", group, stream), ts, stream, message)

	return nil
}

func (p *Provider) AppLogs(name string, opts structs.LogsOptions) (io.ReadCloser, error) {
	r, w := io.Pipe()

	go subscribeLogs(p.Context(), w, name, opts)

	return r, nil
}

func (p *Provider) BuildLogs(app, id string, opts structs.LogsOptions) (io.ReadCloser, error) {
	b, err := p.BuildGet(app, id)
	if err != nil {
		return nil, err
	}

	switch b.Status {
	case "running":
		return p.ProcessLogs(app, b.Process, opts)
	default:
		u, err := url.Parse(b.Logs)
		if err != nil {
			return nil, err
		}

		switch u.Scheme {
		case "object":
			return p.ObjectFetch(u.Hostname(), u.Path)
		default:
			return nil, fmt.Errorf("unable to read logs for build: %s", id)
		}
	}
}

func (p *Provider) ProcessLogs(app, pid string, opts structs.LogsOptions) (io.ReadCloser, error) {
	ps, err := p.ProcessGet(app, pid)
	if err != nil {
		return nil, err
	}

	stream := fmt.Sprintf("%s/service/%s/%s", app, ps.Name, pid)

	r, w := io.Pipe()

	ctx, cancel := context.WithCancel(p.Context())

	go subscribeLogs(ctx, w, stream, opts)
	go p.watchForProcessTermination(ctx, app, pid, cancel)

	return r, nil
}

func (p *Provider) SystemLogs(opts structs.LogsOptions) (io.ReadCloser, error) {
	r, w := io.Pipe()

	go subscribeLogs(p.Context(), w, "rack", opts)

	return r, nil
}

func subscribeLogs(ctx context.Context, w io.WriteCloser, stream string, opts structs.LogsOptions) {
	defer w.Close()

	ch := make(chan logstore.Log, 1000)

	sctx, cancel := context.WithCancel(ctx)
	defer cancel()

	since := time.Now().UTC().Add(-1 * helpers.DefaultDuration(opts.Since, 0))
	follow := helpers.DefaultBool(opts.Follow, true)

	logs.Subscribe(sctx, ch, stream, since, follow)

	for {
		select {
		case <-ctx.Done():
			return
		case l, ok := <-ch:
			if !ok {
				return
			}
			prefix := ""
			if helpers.DefaultBool(opts.Prefix, false) {
				prefix = fmt.Sprintf("%s %s ", l.Timestamp.Format(time.RFC3339), l.Prefix)
			}
			if _, err := fmt.Fprintf(w, "%s%s\n", prefix, l.Message); err != nil {
				return
			}
		}
	}
}
