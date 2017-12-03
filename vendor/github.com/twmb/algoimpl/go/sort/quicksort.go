package sort

// currently chooses pivot to be s[to].
func partition(s Sortable, from, to int) int {
	i := from
	for j := from; j < to-1; j++ {
		if s.Less(j, to-1) {
			s.Swap(i, j)
			i++
		}
	}
	s.Swap(to-1, i)
	return i
}

// TODO: median of three pivot,
//       tail call so only one recursive call
//       insertion sort for small values
//       wtf are the go authors doing with heapsort
//       dual pivot
func quickSort(s Sortable, from, to int) {
	if to-from > 1 {
		pivot := partition(s, from, to)
		quickSort(s, from, pivot)
		quickSort(s, pivot+1, to)
	}
}

func QuickSort(s Sortable) {
	quickSort(s, 0, s.Len())
}
