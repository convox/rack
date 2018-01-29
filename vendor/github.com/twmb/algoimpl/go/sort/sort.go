// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// Package sort is my own implementation of different sort functions
// The package will be similar, if not exactly the same as the go provided
// sort package. I am using that to reference where I need to improve my code.
//
// Another reason for this is that, really, you need Len, Less and Swap if you
// are to make a generic sort package
//
// This package will contain all sort algorithms that operate without needing
// to access a specific element (in essence, all in place sorts)
package sort

type Sortable interface {
	// Len is the number of elements in the collection
	Len() int
	// Less returns whether the element at index i should
	// sort before the element at index j
	Less(i, j int) bool
	// Swaps the elements at indices i and j
	Swap(i, j int)
}

// Insertion sort on Sortable type
// Does not have start and end indices yet, like the Go authors of sort
func InsertionSort(stuff Sortable) {
	for j := 1; j < stuff.Len(); j++ { // from the second spot to the last
		for i := j; i > 0 && stuff.Less(i, i-1); i-- { // while left is larger
			stuff.Swap(i, i-1) // slide right one position
		}
	}
}
