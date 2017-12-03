package controllers

import (
	"net/http"
	"strconv"

	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/structs"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

func init() {
}

func InstancesKeyroll(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	err := Provider.InstanceKeyroll()
	if err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}

func InstancesList(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	instances, err := Provider.InstanceList()
	if err != nil {
		return httperr.Server(err)
	}

	return RenderJson(rw, instances)
}

func InstanceSSH(ws *websocket.Conn) *httperr.Error {
	vars := mux.Vars(ws.Request())
	id := vars["id"]

	opts := structs.InstanceShellOptions{}

	opts.Command = ws.Request().Header.Get("Command")
	opts.Terminal = ws.Request().Header.Get("Terminal")

	var err error

	if opts.Terminal != "" {
		opts.Height, err = strconv.Atoi(ws.Request().Header.Get("Height"))
		if err != nil {
			return httperr.Server(err)
		}
		opts.Width, err = strconv.Atoi(ws.Request().Header.Get("Width"))
		if err != nil {
			return httperr.Server(err)
		}
	}

	if err := Provider.InstanceShell(id, ws, opts); err != nil {
		return httperr.Server(err)
	}

	return nil
}

func InstanceTerminate(rw http.ResponseWriter, r *http.Request) *httperr.Error {
	id := mux.Vars(r)["id"]

	if err := Provider.InstanceTerminate(id); err != nil {
		return httperr.Server(err)
	}

	return RenderSuccess(rw)
}
