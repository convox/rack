package stdcli

import (
	"fmt"
	"strings"
)

type Info struct {
	Context *Context
	Rows    []InfoRow
}

type InfoRow struct {
	Header string
	Value  string
}

func (i *Info) Add(header, value string) {
	i.Rows = append(i.Rows, InfoRow{Header: header, Value: value})
}

func (i *Info) Print() error {
	f := i.formatString()

	for _, r := range i.Rows {
		value := strings.Replace(r.Value, "\n", fmt.Sprintf(fmt.Sprintf("\n%%%ds  ", i.headerWidth()), ""), -1)
		i.Context.Writef(f, r.Header, value)
	}

	return nil
}

func (i *Info) formatString() string {
	return fmt.Sprintf("<h1>%%-%ds</h1>  <value>%%s</value>\n", i.headerWidth())
}

func (i *Info) headerWidth() int {
	w := 0

	for _, r := range i.Rows {
		if len(r.Header) > w {
			w = len(r.Header)
		}
	}

	return w
}
