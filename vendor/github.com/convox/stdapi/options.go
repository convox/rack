package stdapi

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"time"
)

func UnmarshalOptions(r *http.Request, opts interface{}) error {
	v := reflect.ValueOf(opts).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		u := v.Field(i)
		d := f.Tag.Get("default")

		if n := f.Tag.Get("header"); n != "" {
			v := coalesce(r.Header.Get(n), d)
			if err := unmarshalValue(r, u, v); err != nil {
				return err
			}
			continue
		}

		if n := f.Tag.Get("param"); n != "" {
			v := coalesce(r.FormValue(n), d)
			if err := unmarshalValue(r, u, v); err != nil {
				return err
			}
			continue
		}

		if n := f.Tag.Get("query"); n != "" {
			v := coalesce(r.URL.Query().Get(n), d)
			if err := unmarshalValue(r, u, v); err != nil {
				return err
			}
			continue
		}
	}

	return nil
}

func unmarshalValue(r *http.Request, u reflect.Value, v string) error {
	if v == "" {
		return nil
	}

	switch t := u.Interface().(type) {
	case *bool:
		b := v == "true"
		u.Set(reflect.ValueOf(&b))
	case *int:
		i, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		u.Set(reflect.ValueOf(&i))
	case *string:
		u.Set(reflect.ValueOf(&v))
	case *time.Duration:
		d, err := time.ParseDuration(v)
		if err != nil {
			return err
		}
		u.Set(reflect.ValueOf(&d))
	case map[string]string:
		uv, err := url.ParseQuery(v)
		if err != nil {
			return err
		}
		m := map[string]string{}
		for k := range uv {
			m[k] = uv.Get(k)
		}
		u.Set(reflect.ValueOf(m))
	default:
		return fmt.Errorf("unknown param type: %T", t)
	}

	return nil
}
