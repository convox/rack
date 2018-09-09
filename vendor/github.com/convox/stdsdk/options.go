package stdsdk

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"reflect"
	"time"
)

type Files map[string][]byte
type Headers map[string]string
type Params map[string]interface{}
type Query map[string]interface{}

type RequestOptions struct {
	Body    io.Reader
	Files   Files
	Headers Headers
	Params  Params
	Query   Query
}

func (o *RequestOptions) Querystring() (string, error) {
	u, err := marshalValues(o.Query)
	if err != nil {
		return "", err
	}

	return u.Encode(), nil
}

func (o *RequestOptions) Content() (io.Reader, string, error) {
	if o.Body != nil && len(o.Files) > 0 {
		return nil, "", fmt.Errorf("cannot specify both Body and Files")
	}

	if o.Body != nil && len(o.Params) > 0 {
		return nil, "", fmt.Errorf("cannot specify both Body and Params")
	}

	if o.Body == nil && len(o.Files) == 0 && len(o.Params) == 0 {
		return nil, "application/octet-stream", nil
	}

	if o.Body != nil {
		return o.Body, "application/octet-stream", nil
	}

	uv, err := marshalValues(o.Params)
	if err != nil {
		return nil, "", err
	}

	if len(o.Files) > 0 {
		var buf bytes.Buffer

		w := multipart.NewWriter(&buf)

		for name, data := range o.Files {
			part, err := w.CreateFormFile(name, "binary-data")
			if err != nil {
				return nil, "", err
			}

			if _, err := part.Write(data); err != nil {
				return nil, "", err
			}
		}

		for k := range uv {
			w.WriteField(k, uv.Get(k))
		}

		if err := w.Close(); err != nil {
			return nil, "", err
		}

		return &buf, w.FormDataContentType(), nil
	}

	return bytes.NewReader([]byte(uv.Encode())), "application/x-www-form-urlencoded", nil
}

func MarshalOptions(opts interface{}) (RequestOptions, error) {
	ro := RequestOptions{
		Headers: Headers{},
		Params:  Params{},
		Query:   Query{},
	}

	v := reflect.ValueOf(opts)
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		if n := f.Tag.Get("header"); n != "" {
			if u, ok := marshalValue(v.Field(i)); ok {
				ro.Headers[n] = u
			}
		}

		if n := f.Tag.Get("param"); n != "" {
			if u, ok := marshalValue(v.Field(i)); ok {
				ro.Params[n] = u
			}
		}

		if n := f.Tag.Get("query"); n != "" {
			if u, ok := marshalValue(v.Field(i)); ok {
				ro.Query[n] = u
			}
		}
	}

	return ro, nil
}

func marshalValue(f reflect.Value) (string, bool) {
	if f.IsNil() {
		return "", false
	}

	v := f.Interface()

	if f.Kind() == reflect.Ptr {
		v = f.Elem().Interface()
	}

	switch t := v.(type) {
	case bool:
		return fmt.Sprintf("%t", t), true
	case int, int64:
		return fmt.Sprintf("%d", t), true
	case string:
		return t, true
	case time.Duration:
		return t.String(), true
	case time.Time:
		return t.Format("20060102.150405.000000000"), true
	case map[string]string:
		uv := url.Values{}
		for k, v := range t {
			uv.Add(k, v)
		}
		return uv.Encode(), true
	default:
		return "", false
	}

	return "", true
}

func marshalValues(vv map[string]interface{}) (url.Values, error) {
	u := url.Values{}

	for k, v := range vv {
		switch t := v.(type) {
		case bool:
			u.Set(k, fmt.Sprintf("%t", t))
		case int, int64:
			u.Set(k, fmt.Sprintf("%d", t))
		case string:
			u.Set(k, t)
		case []string:
			for _, s := range t {
				u.Add(k, s)
			}
		case time.Duration:
			u.Set(k, t.String())
		case map[string]string:
			uv := url.Values{}
			for kk, vv := range t {
				uv.Set(kk, vv)
			}
			u.Set(k, uv.Encode())
		default:
			return nil, fmt.Errorf("unknown param type: %T", t)
		}
	}

	return u, nil
}
