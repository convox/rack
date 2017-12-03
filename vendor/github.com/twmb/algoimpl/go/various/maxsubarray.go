// Implements the maximum subarray algorithm on a slice of ints
package various

func maxCrossingSubarray(array []int, from, to int) (li, ri, sum int) {
	mid := (from + to) / 2
	// left index, left side's sum, new running sum
	li, lsum, nsum := mid-1, array[mid-1], array[mid-1]
	for n := li - 1; n >= from; n-- {
		nsum += array[n]
		if nsum > lsum {
			lsum = nsum
			li = n
		}
	}
	ri, rsum, nsum := mid, array[mid], array[mid]
	for n := ri + 1; n < to; n++ {
		nsum += array[n]
		if nsum > rsum {
			rsum = nsum
			ri = n
		}
	}
	return li, ri + 1, lsum + rsum // one after last valid index
}

func MaxSubarrayRecursive(array []int, from, to int) (li, ri, sum int) {
	if from >= to-1 {
		if to-from == 0 {
			return from, to, 0
		}
		return from, to, array[from]
	} else {
		lli, lri, lv := MaxSubarrayRecursive(array, from, (from+to)/2)
		rli, rri, rv := MaxSubarrayRecursive(array, (from+to)/2, to)
		cli, cri, cv := maxCrossingSubarray(array, from, to)
		if lv > rv && lv > cv {
			return lli, lri, lv // left's left index, right index, sum
		} else if rv > lv && rv > cv {
			return rli, rri, rv // right's left index, right index, sum
		} else {
			return cli, cri, cv // crossing left index, right index, sum
		}
	}
}

// iterative
func MaxSubarray(array []int, from, to int) (li, ri, sum int) {
	if to-from <= 1 {
		if to-from == 1 {
			return from, to, array[from]
		}
		return from, to, 0
	}
	hli, hri, hsum := from, from, array[from] // subarray right now
	bli, bri, bsum := from, from, array[from] // best so far

	for here := from + 1; here < to; here++ {
		if hsum+array[here] > array[here] { // if subarray now is still
			// higher than when started
			hri = here          // slide right index right one more
			hsum += array[here] // record new sum
		} else { // else subarray lost, begin tracking from local valley
			hli, hri, hsum = here, here, array[here]
		}
		if hsum > bsum { // record highest sum while it is net positive
			bli, bri, bsum = hli, hri, hsum
		}
	}
	return bli, bri + 1, bsum // here's right index is absolute, want one past
}

func MaxSubarray2(a []int) (max []int, sum int) {
	if len(a) < 1 {
		return a, 0
	}
	sum = a[0]
	sumHere := sum
	startMaxHere := 0
	maxHere := a[:1]
	max = maxHere
	for i := 1; i < len(a); i = i + 1 {
		if sumHere+a[i] < a[i] {
			sumHere = a[i]
			startMaxHere = i
		} else {
			sumHere += a[i]
		}
		maxHere = a[startMaxHere : i+1]
		if sumHere > sum {
			sum = sumHere
			max = maxHere
		}
	}
	return
}
