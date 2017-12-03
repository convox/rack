package dupsort

import "testing"

type Ints []int

func (p Ints) Len() int { return len(p) }
func (p Ints) Less(i, j interface{}) bool {
	l, okl := i.(int)
	r, okr := j.(int)
	if !okl || !okr {
		return false
	}
	return l < r
}
func (p Ints) At(i int) interface{} { return p[i] }
func (p Ints) Set(i int, n interface{}) {
	v, ok := n.(int)
	if ok {
		p[i] = v
	}
}
func (p Ints) New(i int) DupSortable {
	ints := make([]int, i)
	return Ints(ints)
}

func TestMergeSort(t *testing.T) {
	// 100% line coverage
	tests := []struct {
		In, Want Ints
	}{
		{Ints([]int{21, -10, 54, 0, 1098309}), Ints([]int{-10, 0, 21, 54, 1098309})},
		{Ints([]int{}), Ints([]int{})},
		{Ints([]int{4, 3, 2, 1, 2, 3}), Ints([]int{1, 2, 2, 3, 3, 4})},
	}

	for _, test := range tests {
		gotI := MergeSort(test.In, 0, len(test.In))
		got := gotI.(Ints)
		failed := false
		for i, v := range got {
			if v != test.Want[i] {
				t.Errorf("MergeSort, position %v, sortval %v, supposed to be %v", i, v, test.Want[i])
				failed = true
			}
		}
		if failed {
			t.Errorf("Failing Ints: %v != %v", test.In, test.Want)
		}
	}
}
