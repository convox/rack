package various

import (
	"math/rand"
	"time"
)

// returns the final pivot index
func partition(slice []int) int {
	pi := rand.Intn(len(slice))
	n := len(slice) - 1                       // convenience
	slice[n], slice[pi] = slice[pi], slice[n] // swap pivot to end
	b := 0                                    // boundry between larger and smaller elements. demarcates first element index larger than p[i]
	for i := 0; i < n; i++ {
		if slice[i] < slice[n] {
			slice[i], slice[b] = slice[b], slice[i] // smaller element, swap boundry element (large) with this smaller element
			b++                                     // and increment the boundry to point past the smaller element
		}
	}
	pi = b
	slice[n], slice[pi] = slice[pi], slice[n] // swap pivot to boundry between larger and smaller
	return pi                                 // and return the index
}

func selectOrder(i int, slice []int) int {
	if len(slice) == 1 {
		return slice[0]
	} else {
		q := partition(slice)
		if i == q {
			return slice[q]
		} else if i < q {
			return selectOrder(i, slice[:q])
		}
		return selectOrder(i-(q+1), slice[q+1:])
	}
}

// Returns the ith smallest element from a slice of integers.
// Runs in expected O(n) time, O(n^2) worst time (unlikely)
func SelectOrder(i int, slice []int) int {
	rand.Seed(time.Now().Unix())
	cslice := make([]int, len(slice)) // copy
	copy(cslice, slice[:])
	return selectOrder(i, cslice)
}
