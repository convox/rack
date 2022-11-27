package stdapi

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

func UnmarshalOptions(r *http.Request, opts interface{}) error {
	v := reflect.ValueOf(opts).Elem()
	t := v.Type()

	r.ParseMultipartForm(8192)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		u := v.Field(i)
		d := f.Tag.Get("default")

		if n := f.Tag.Get("header"); n != "" {
			if v, ok := fetchHeader(r, n, d); ok {
				if err := unmarshalValue(r, u, v); err != nil {
					return errors.WithStack(err)
				}
			}
			continue
		}

		if n := f.Tag.Get("param"); n != "" {
			if v, ok := fetchForm(r, n, d); ok {
				if err := unmarshalValue(r, u, v); err != nil {
					return errors.WithStack(err)
				}
			}
			continue
		}

		if n := f.Tag.Get("query"); n != "" {
			if v, ok := fetchQuery(r, n, d); ok {
				if err := unmarshalValue(r, u, v); err != nil {
					return errors.WithStack(err)
				}
			}
			continue
		}
	}

	return nil
}

func fetchForm(r *http.Request, name string, def string) (string, bool) {
	if v, ok := r.Form[name]; ok {
		return v[0], true
	}
	if def != "" {
		return def, true
	}
	return "", false
}

func fetchHeader(r *http.Request, name string, def string) (string, bool) {
	if v, ok := r.Header[name]; ok {
		return v[0], true
	}
	if def != "" {
		return def, true
	}
	return "", false
}

func fetchQuery(r *http.Request, name string, def string) (string, bool) {
	if v, ok := r.URL.Query()[name]; ok {
		return v[0], true
	}
	if def != "" {
		return def, true
	}
	return "", false
}

func unmarshalValue(r *http.Request, u reflect.Value, v string) error {
	switch t := u.Interface().(type) {
	case *bool:
		b := v == "true"
		u.Set(reflect.ValueOf(&b))
	case *int:
		i, err := strconv.Atoi(v)
		if err != nil {
			return errors.WithStack(err)
		}
		u.Set(reflect.ValueOf(&i))
	case *int64:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return errors.WithStack(err)
		}
		u.Set(reflect.ValueOf(&i))
	case *string:
		u.Set(reflect.ValueOf(&v))
	case *time.Duration:
		d, err := time.ParseDuration(v)
		if err != nil {
			return errors.WithStack(err)
		}
		u.Set(reflect.ValueOf(&d))
	case *time.Time:
		tt, err := time.Parse("20060102.150405.000000000", v)
		if err != nil {
			return errors.WithStack(err)
		}
		u.Set(reflect.ValueOf(&tt))
	case []string:
		ss := strings.Split(v, ",")
		u.Set(reflect.ValueOf(ss))
	case *[]string:
		ss := strings.Split(v, ",")
		u.Set(reflect.ValueOf(&ss))
	case map[string]string:
		uv, err := url.ParseQuery(v)
		if err != nil {
			return errors.WithStack(err)
		}
		m := map[string]string{}
		for k := range uv {
			m[k] = uv.Get(k)
		}
		u.Set(reflect.ValueOf(m))
	default:
		return errors.WithStack(fmt.Errorf("unknown param type: %T", t))
	}

	return nil
}
