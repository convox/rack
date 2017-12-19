package controllers

import (
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"golang.org/x/net/websocket"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/provider"
	"github.com/convox/rack/structs"
)

func SystemShow(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	rack, err := Provider.SystemGet()
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, rack)
}

func SystemProcesses(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	all := r.URL.Query().Get("all")

	ps, err := Provider.SystemProcesses(structs.SystemProcessesOptions{
		All: (all == "true"),
	})
	if provider.ErrorNotFound(err) {
		return httperr.NotFound(err)
	}
	if err != nil {
		return httperr.Server(err)
	}

	sort.Slice(ps, ps.Less)

	return RenderJson(rw, ps)
}

func SystemUpdate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	opts := structs.SystemUpdateOptions{}

	// update based on form input
	if cc := GetForm(r, "count"); cc != "" {
		c, err := strconv.Atoi(cc)
		if err != nil {
			return httperr.Errorf(403, "count must be numeric")
		}

		switch {
		case os.Getenv("AUTOSCALE") == "true":
			return httperr.Errorf(403, "scaling count prohibited when autoscale enabled")
		case c == -1:
			// -1 indicates no change
		case c <= 2:
			return httperr.Errorf(403, "count must be greater than 2")
		default:
			opts.InstanceCount = c
		}
	}

	if t := GetForm(r, "type"); t != "" {
		opts.InstanceType = t
	}

	if v := GetForm(r, "version"); v != "" {
		opts.Version = v
	}

	if err := Provider.SystemUpdate(opts); err != nil {
		return httperr.Server(err)
	}

	s, err := Provider.SystemGet()
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, s)
}

func SystemCapacity(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	capacity, err := Provider.CapacityGet()
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, capacity)
}

// SystemLogs returns the logs for the Rack
func SystemLogs(ws *websocket.Conn) *httperr.Error {
	header := ws.Request().Header

	var err error

	follow := true
	if header.Get("Follow") == "false" {
		follow = false
	}

	since := 2 * time.Minute
	if s := header.Get("Since"); s != "" {
		since, err = time.ParseDuration(s)
		if err != nil {
			return httperr.Errorf(403, "Invalid duration %s", s)
		}
	}

	r, err := Provider.SystemLogs(structs.LogsOptions{
		Filter: header.Get("Filter"),
		Follow: follow,
		Since:  time.Now().Add(-1 * since),
	})
	if err != nil {
		return httperr.Server(err)
	}

	io.Copy(ws, r)

	return nil
}

func SystemReleases(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	releases, err := Provider.SystemReleases()
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, releases)
}
