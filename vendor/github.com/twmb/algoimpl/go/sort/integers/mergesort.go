// Implements merge sort on []ints.
// Knowing that the underlying type is a slice allows for using channels.
// This also allows for using goroutines.
package integers

func mergeCombine(lch, rch <-chan int, tch chan<- int) {
	lv, lopen := <-lch
	rv, ropen := <-rch
	for lopen && ropen {
		if lv < rv {
			tch <- lv
			lv, lopen = <-lch
		} else {
			tch <- rv
			rv, ropen = <-rch
		}
	}
	for lopen {
		tch <- lv
		lv, lopen = <-lch
	}
	for ropen {
		tch <- rv
		rv, ropen = <-rch
	}
	close(tch)
}

// This function takes a slice to be sorted, a range to sort
// and a channel to send in-order ints to.
func MergeSort(me []int, from, to int, tch chan<- int) {
	if from < to-1 {
		lch, rch := make(chan int), make(chan int)
		go MergeSort(me, from, (from+to)/2, lch)
		go MergeSort(me, (from+to)/2, to, rch)
		mergeCombine(lch, rch, tch)
	} else {
		for i := from; i < to; i++ {
			tch <- me[i]
		}
		close(tch)
	}
}
