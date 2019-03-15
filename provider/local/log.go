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

func (p *Provider) Log(app, pid string, ts time.Time, message string) error {
	logs.Append(app, pid, ts, message)

	return nil
}

func (p *Provider) AppLogs(name string, opts structs.LogsOptions) (io.ReadCloser, error) {
	r, w := io.Pipe()

	go p.subscribeLogs(w, logs.Group(name).Subscribe, opts)

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
	r, w := io.Pipe()

	go p.subscribeLogs(w, logs.Group(app).Stream(pid).Subscribe, opts)

	return r, nil
}

func (p *Provider) SystemLogs(opts structs.LogsOptions) (io.ReadCloser, error) {
	r, w := io.Pipe()

	go p.subscribeLogs(w, logs.Group("rack").Subscribe, opts)

	return r, nil
}

func (p *Provider) subscribeLogs(w io.WriteCloser, sub logstore.Subscribe, opts structs.LogsOptions) {
	defer w.Close()

	ch := make(chan logstore.Log)

	ctx, cancel := context.WithCancel(p.Context())
	defer cancel()

	go sub(ctx, ch, time.Now().UTC().Add(-1*helpers.DefaultDuration(opts.Since, 0)), helpers.DefaultBool(opts.Follow, true))

	for {
		select {
		case <-p.Context().Done():
			return
		case l, ok := <-ch:
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(w, "%s %s %s\n", l.Timestamp.Format(time.RFC3339), l.Stream, l.Message); err != nil {
				return
			}
		}
	}
}
