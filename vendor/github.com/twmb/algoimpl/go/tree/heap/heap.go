// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// Package sort is my own implementation of different sort functions
// Package heap has functions for creating and abusing a heap.
// It is almost identically the same as containers/heap, so
// just use that.
package heap

import (
	"sort"
)

// Any type that implements Interface may be used as a
// max-tree. A tree must be either first Init()'d or build from scratch.
// The tree functions will panic if called inappropriately (as in, you
// cannot call Pop on an empty tree).
//
// This interface embeds sort.Interface, meaning elements
// can be compared and swapped.
//
// Note that Push and Pop are for this package to call. To add or remove
// elements from a tree, use tree.Push and tree.Pop.
type Interface interface {
	sort.Interface
	// Adds a value to the end of the collection.
	Push(val interface{})
	// Removes the value at the end of the collection.
	Pop() interface{}
}

// Creates a max heap out of an unorganized Interface collection.
// Runs in O(n) time, where n = h.Len().
func Init(h Interface) {
	for i := h.Len()/2 - 1; i >= 0; i-- { // start at first non leaf (equiv. to parent of last leaf)
		shuffleDown(h, i, h.Len())
	}
}

// Removes and returns the maximum of the heap and reorganizes.
// The complexity is O(lg n).
func Pop(h Interface) interface{} {
	n := h.Len() - 1
	h.Swap(n, 0)
	shuffleDown(h, 0, n)
	return h.Pop()
}

// This function will push a new value into a priority queue.
func Push(h Interface, val interface{}) {
	h.Push(val)
	shuffleUp(h, h.Len()-1)
}

// Removes and returns the element at index i
func Remove(h Interface, i int) (v interface{}) {
	n := h.Len() - 1
	if n != i {
		h.Swap(n, i)
		shuffleDown(h, i, n)
		shuffleUp(h, i)
	}
	return h.Pop()
}

// Shuffles a smaller value at index i in a heap
// down to the appropriate spot. Complexity is O(lg n).
func shuffleDown(heap Interface, i, end int) {
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

// Shuffles a larger value in a heap at index i
// up to the appropriate spot. Complexity is O(lg n).
func shuffleUp(heap Interface, i int) {
	for {
		pi := (i - 1) / 2 // (i + 1) / 2 - 1, parent
		if i == pi || !heap.Less(pi, i) {
			break
		}
		heap.Swap(pi, i)
		i = pi
	}
}
