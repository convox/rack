package stdcli

import "fmt"

type Info struct {
	Rows []InfoRow
}

type InfoRow struct {
	Name  string
	Value string
}

func NewInfo() *Info {
	return &Info{Rows: []InfoRow{}}
}

func (i *Info) Add(name, value string) {
	i.Rows = append(i.Rows, InfoRow{
		Name:  name,
		Value: value,
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
		Writef(fmt.Sprintf("<header>%%-%ds</header>  %%s\n", longest), r.Name, r.Value)
	}
}
