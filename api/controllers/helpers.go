package controllers

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sort"
	"time"

	"github.com/convox/rack/helpers"
	"golang.org/x/net/websocket"
)

func coalesce(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}

	return ""
}

func readChannel(r io.Reader, datach chan string, donech chan error) {
	buf := make([]byte, 10*1024)

	for {
		n, err := r.Read(buf)
		if err != nil {
			if err == io.EOF {
				donech <- nil
			} else {
				donech <- err
			}
			return
		}

		datach <- string(buf[0:n])
	}
}

type sortableSlice interface {
	Less(int, int) bool
}

func sortSlice(s sortableSlice) {
	sort.Slice(s, s.Less)
}

func streamWebsocket(ws *websocket.Conn, r io.ReadCloser) error {
	defer r.Close()

	datach := make(chan string)
	donech := make(chan error)
	tick := time.Tick(1 * time.Second)

	go readChannel(r, datach, donech)

	for {
		select {
		case <-tick:
			// check for closed connection
			if _, err := ws.Write([]byte{}); err != nil {
				return nil
			}
		case data := <-datach:
			ws.Write([]byte(data))
		case err := <-donech:
			return err
		}
	}

	return nil
}

func unmarshalOptions(r *http.Request, opts interface{}) error {
	v := reflect.ValueOf(opts).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		u := v.Field(i)

		if n := f.Tag.Get("param"); n != "" {
			if v := r.FormValue(n); v != "" {
				if err := unmarshalValue(r, n, u, v); err != nil {
					return err
				}
			}
		}

		if n := f.Tag.Get("query"); n != "" {
			if v := r.URL.Query().Get(n); v != "" {
				if err := unmarshalValue(r, n, u, v); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func unmarshalValue(r *http.Request, param string, u reflect.Value, v string) error {
	switch t := u.Interface().(type) {
	case *bool:
		b := v == "true"
		u.Set(reflect.ValueOf(&b))
	case *string:
		u.Set(reflect.ValueOf(&v))
	case *time.Time:
		tv, err := time.Parse(helpers.SortableTime, v)
		if err != nil {
			return err
		}
		u.Set(reflect.ValueOf(&tv))
	default:
		return fmt.Errorf("unknown param type: %T", t)
	}

	return nil
}
