package controllers

import (
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"time"

	"github.com/convox/rack/helpers"
)

type sortableSlice interface {
	Less(int, int) bool
}

func sortSlice(s sortableSlice) {
	sort.Slice(s, s.Less)
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
