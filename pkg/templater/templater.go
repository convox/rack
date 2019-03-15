package templater

import (
	"bytes"
	"html/template"

	"github.com/gobuffalo/packr"
)

type Templater struct {
	box     packr.Box
	helpers template.FuncMap
}

func New(box packr.Box, helpers template.FuncMap) *Templater {
	// fmt.Printf("box.List() = %+v\n", box.List())

	return &Templater{
		box:     box,
		helpers: helpers,
	}
}

func (t *Templater) Render(name string, params interface{}) ([]byte, error) {
	ts := template.New("").Funcs(t.helpers)

	tdata, err := t.box.MustString(name)
	if err != nil {
		return nil, err
	}

	if _, err := ts.Parse(tdata); err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	if err := ts.Execute(&buf, params); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
