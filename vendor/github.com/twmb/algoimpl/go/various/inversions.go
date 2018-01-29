package various

func inversionsCombine(left, right, combined []int) int {
	inversions := 0
	k, li, ri := 0, 0, 0 // index in combined array
	for ; li < len(left) && ri < len(right); k++ {
		if left[li] < right[ri] {
			combined[k] = left[li]
			li++
		} else { // right less than left
			combined[k] = right[ri]
			inversions += len(left) - li // if a right element is larger than a left,
			ri++                         // then it is larger than every element remaining on the left
		}
	}
	for ; li < len(left); li, k = li+1, k+1 {
		combined[k] = left[li]
	}
	for ; ri < len(right); ri, k = ri+1, k+1 {
		combined[k] = right[ri]
	}
	return inversions
}

// performs a mergesort while counting inversions
func inversionsCount(array, buffer []int) int {
	if len(array) <= 1 {
		return 0
	}
	cleft := inversionsCount(buffer[:len(array)/2], array[:len(array)/2])
	cright := inversionsCount(buffer[len(array)/2:], array[len(array)/2:])
	ccross := inversionsCombine(array[:len(array)/2], array[len(array)/2:], buffer)
	return cleft + ccross + cright
}

// Inversions will return the number of inversions in a given input integer array.
// An inversion is when a smaller number appears after a larger number.
// For example, [1,3,5,2,4,6] has three inversions: [3,2], [5,2] and [5,4].
// Runs in O(n lg n) time, where n is the size of the input.
func Inversions(array []int) int {
	buffer0 := make([]int, len(array))
	buffer1 := make([]int, len(array))
	copy(buffer0, array[:])
	copy(buffer1, array[:])
	count := inversionsCount(buffer0, buffer1)
	return count
}
