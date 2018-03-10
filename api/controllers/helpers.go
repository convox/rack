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

func readChannel(r io.Reader, ch chan []byte) {
	buf := make([]byte, 10*1024)

	for {
		n, err := r.Read(buf)
		if err != nil {
			ch <- nil
			return
		}

		ch <- buf[0:n]
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

	ch := make(chan []byte)
	tick := time.Tick(1 * time.Second)

	go readChannel(r, ch)

	for {
		select {
		case <-tick:
			// check for closed connection
			if _, err := ws.Write([]byte{}); err != nil {
				return nil
			}
		case data := <-ch:
			ws.Write(data)
		}
	}

	return nil
}

func unmarshalOptions(r *http.Request, opts interface{}) error {
	v := reflect.ValueOf(opts).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		n := f.Tag.Get("param")

		if n != "" {
			if fv := r.FormValue(n); fv != "" {
				u := v.Field(i)

				switch u.Interface().(type) {
				case *string:
					u.Set(reflect.ValueOf(&fv))
				case *time.Time:
					t, err := time.Parse(helpers.SortableTime, fv)
					if err != nil {
						return err
					}
					u.Set(reflect.ValueOf(&t))
				default:
					return fmt.Errorf("unknown param type: %T", t)
				}
			}
		}
	}

	return nil
}
