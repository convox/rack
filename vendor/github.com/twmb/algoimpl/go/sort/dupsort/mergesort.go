// Implements merge sort on the abstract data type DupSortable
package dupsort

func mergeCombine(l, r DupSortable) DupSortable {
	combined := l.New(r.Len() + l.Len())
	li, ri, ci := 0, 0, 0
	for ; li < l.Len() && ri < r.Len(); ci++ {
		if l.Less(l.At(li), r.At(ri)) {
			combined.Set(ci, l.At(li))
			li++
		} else {
			combined.Set(ci, r.At(ri))
			ri++
		}
	}
	for ; li < l.Len(); ci, li = ci+1, li+1 {
		combined.Set(ci, l.At(li))
	}
	for ; ri < r.Len(); ci, ri = ci+1, ri+1 {
		combined.Set(ci, r.At(ri))
	}
	return combined
}

func MergeSort(me DupSortable, from, to int) DupSortable {
	if from < to-1 {
		left := MergeSort(me, from, (from+to)/2)
		right := MergeSort(me, (from+to)/2, to)
		combined := mergeCombine(left, right)
		return combined
	}
	ele := me.New(to - from)
	if to-from > 0 {
		ele.Set(0, me.At(from))
	}
	return ele
}
