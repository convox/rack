package stdcli

import (
	"fmt"
	"strings"
	"unicode"
)

type Table struct {
	Headers     []string
	Rows        [][]string
	SkipHeaders bool
}

func NewTable(headers ...string) *Table {
	return &Table{Headers: headers}
}

func (t *Table) AddRow(values ...string) {
	t.Rows = append(t.Rows, values)
}

func (t *Table) Print() {
	if !t.SkipHeaders {
		t.printHeaders(t.Headers)
	}

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

func (t *Table) printHeaders(values []string) {
	line := fmt.Sprintf(t.formatString(), interfaceSlice(values)...)
	line = strings.TrimRightFunc(line, unicode.IsSpace) + "\n"

	Writef("<header>%s</header>", line)
}

func (t *Table) printValues(values []string) {
	line := fmt.Sprintf(t.formatString(), interfaceSlice(values)...)
	line = strings.TrimRightFunc(line, unicode.IsSpace) + "\n"

	Write([]byte(line))
}

func interfaceSlice(ss []string) []interface{} {
	is := make([]interface{}, len(ss))

	for i, s := range ss {
		is[i] = s
	}

	return is
}
