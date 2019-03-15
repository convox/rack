package api

type Sortable interface {
	Less(int, int) bool
}
