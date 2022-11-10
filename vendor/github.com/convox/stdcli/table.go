package stdcli

import (
	"fmt"
	"strings"
)

type Table struct {
	Columns []string
	Context *Context
	Rows    [][]string
}

func (t *Table) AddRow(row ...string) {
	t.Rows = append(t.Rows, row)
}

func (t *Table) Print() error {
	f := t.formatString()

	t.Context.Writef(fmt.Sprintf("<h1>%s</h1>\n", f), interfaceSlice(t.Columns)...)

	for _, r := range t.Rows {
		t.Context.Writef(fmt.Sprintf("<value>%s</value>\n", f), interfaceSlice(r)...)
	}

	return nil
}

func (t *Table) formatString() string {
	f := []string{}

	ws := t.widths()

	for _, w := range ws {
		f = append(f, fmt.Sprintf("%%-%ds", w))
	}

	return strings.Join(f, "  ")
}

func (t *Table) widths() []int {
	w := make([]int, len(t.Columns))

	for i, c := range t.Columns {
		w[i] = len(stripTag(c))

		for _, r := range t.Rows {
			if len(r) > i {
				if lri := len(stripTag(r[i])); lri > w[i] {
					w[i] = lri
				}
			}
		}
	}

	w[len(w)-1] = 0

	return w
}
