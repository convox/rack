package controllers

import "sort"

type sortableSlice interface {
	Less(int, int) bool
}

func sortSlice(s sortableSlice) {
	sort.Slice(s, s.Less)
}
