package stdcli

import (
	"fmt"
	"strings"
)

type Table struct {
	Headers []string
	Rows    [][]string
}

func NewTable(headers ...string) *Table {
	return &Table{Headers: headers}
}

func (t *Table) AddRow(values ...string) {
	t.Rows = append(t.Rows, values)
}

func (t *Table) Print() {
	fs := t.FormatString()

	fmt.Printf(fs, interfaceSlice(t.Headers)...)

	for _, row := range t.Rows {
		fmt.Printf(fs, interfaceSlice(row)...)
	}
}

func (t *Table) FormatString() string {
	longest := make([]int, len(t.Headers))

	for i, header := range t.Headers {
		longest[i] = len(header)
	}

	for _, row := range t.Rows {
		for i, col := range row {
			if l := len(fmt.Sprintf("%v", col)); l > longest[i] {
				longest[i] = l
			}
		}
	}

	parts := make([]string, len(longest))

	for i, l := range longest {
		parts[i] = fmt.Sprintf("%%-%ds", l)
	}

	return strings.Join(parts, "  ") + "\n"
}

func interfaceSlice(ss []string) []interface{} {
	is := make([]interface{}, len(ss))

	for i, s := range ss {
		is[i] = s
	}

	return is
}
