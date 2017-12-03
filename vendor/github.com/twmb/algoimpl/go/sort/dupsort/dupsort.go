// Package dupsort, again, is my own implementation of different sort
// functions. The difference between dupsort and sort is that this package
// contains all algorithms that must actually access an element. This new
// requirement means that dupsort needs a larger interface (by one function).
//
// Generally, an algorithm needs to access an element if it needs to duplicate
// elements. Otherwise they could just be compared and swapped. This means
// that this package includes all algorithms that need to copy what is being
// sorted.
package dupsort

type DupSortable interface {
	// Len is the number of elements in the collection
	Len() int
	// Less returns whether the element i should
	// sort before the element j
	Less(i, j interface{}) bool
	// At accesses the element at index i
	At(i int) interface{}
	// Set sets the value at index i
	Set(i int, val interface{})
	// New returns a new DupSortable of length i
	New(i int) DupSortable
}
