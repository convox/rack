package sort

// Shuffles a smaller value at index i in a heap
// down to the appropriate spot. Complexity is O(lg n).
func shuffleDown(heap Sortable, i, end int) {
	for {
		l := 2*i + 1
		if l >= end { // int overflow? (in go source)
			break
		}
		li := l
		if r := l + 1; r < end && heap.Less(l, r) {
			li = r // 2*i + 2
		}
		if heap.Less(li, i) {
			break
		}
		heap.Swap(li, i)
		i = li
	}
}

// Creates a max heap out of an unorganized Sortable collection.
// Runs in O(n) time.
func buildHeap(stuff Sortable) {
	for i := stuff.Len()/2 - 1; i >= 0; i-- { // start at first non leaf (equiv. to parent of last leaf)
		shuffleDown(stuff, i, stuff.Len())
	}
}

// Runs HeapSort on a Sortable collection.
// Runs in O(n * lg n) time, but amortizes worse than quicksort
func HeapSort(stuff Sortable) {
	buildHeap(stuff)
	for i := stuff.Len() - 1; i > 0; i-- {
		stuff.Swap(0, i) // put max at end
		shuffleDown(stuff, 0, i)
	}
}
