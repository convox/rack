package stdcli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
)

type Table struct {
	Headers []string
	Rows    [][]string

	Output io.Writer
}

func NewTable(headers ...string) *Table {
	return &Table{Headers: headers, Output: os.Stdout}
}

func (t *Table) AddRow(values ...string) {
	t.Rows = append(t.Rows, values)
}

func (t *Table) Print() {
	t.printValues(t.Headers)

	for _, row := range t.Rows {
		t.printValues(row)
	}
}

func (t *Table) formatString() string {
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

func (t *Table) printValues(values []string) {
	line := fmt.Sprintf(t.formatString(), interfaceSlice(values)...)
	line = strings.TrimRightFunc(line, unicode.IsSpace) + "\n"

	fmt.Fprint(t.Output, line)
}

func interfaceSlice(ss []string) []interface{} {
	is := make([]interface{}, len(ss))

	for i, s := range ss {
		is[i] = s
	}

	return is
}
