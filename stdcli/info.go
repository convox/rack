package stdcli

import (
	"fmt"
	"strings"
)

type Info struct {
	Rows []InfoRow
}

type InfoRow struct {
	Name   string
	Values []string
}

func NewInfo() *Info {
	return &Info{Rows: []InfoRow{}}
}

func (i *Info) Add(name string, values ...string) {
	i.Rows = append(i.Rows, InfoRow{
		Name:   name,
		Values: values,
	})
}

func (i *Info) Print() {
	longest := 0

	for _, r := range i.Rows {
		if len(r.Name) > longest {
			longest = len(r.Name)
		}
	}

	for _, r := range i.Rows {
		Writef(fmt.Sprintf("<header>%%-%ds</header>  %%s\n", longest), r.Name, strings.Join(r.Values, "\n"+strings.Repeat(" ", longest+2)))
	}
}
