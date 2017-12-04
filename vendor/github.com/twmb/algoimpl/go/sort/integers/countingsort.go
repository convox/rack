package integers

// Runs counting sort on a slice of ints with the minVal being the minimum value
// in the slice and maxVal being the maximum.
// Has O(n + maxVal - minVal) time complexity, where n is the length of the slice.
func CountingSort(me []int, minVal, maxVal int) (sorted []int) {
	sorted = make([]int, len(me))
	counts := make([]int, maxVal+1-minVal)
	for i := range me {
		counts[me[i]-minVal]++
	}
	for i := 1; i < len(counts); i++ {
		counts[i] += counts[i-1]
	}
	for i := range me {
		sorted[counts[me[i]-minVal]-1] = me[i]
		counts[me[i]-minVal]--
	}
	return
}
