package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/api/httperr"
	"github.com/convox/rack/options"
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

	h := ws.Request().Header

	if v := h.Get("Command"); v != "" {
		opts.Command = options.String(v)
	}

	if v := h.Get("Terminal"); v != "" {
		opts.Terminal = options.String(v)
	}

	if v := h.Get("Height"); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			return httperr.Server(fmt.Errorf("height must be numeric"))
		}
		opts.Height = aws.Int(i)
	}

	if v := h.Get("Width"); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			return httperr.Server(fmt.Errorf("width must be numeric"))
		}
		opts.Width = aws.Int(i)
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
