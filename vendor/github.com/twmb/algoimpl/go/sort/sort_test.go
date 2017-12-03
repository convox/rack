package sort

import "testing"

type Ints []int

func (p Ints) Len() int           { return len(p) }
func (p Ints) Less(i, j int) bool { return p[i] < p[j] }
func (p Ints) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func TestInsertionSort(t *testing.T) {
	// 100% line coverage
	tests := []struct {
		In, Want Ints
	}{
		{Ints([]int{21, -10, 54, 0, 1098309}), Ints([]int{-10, 0, 21, 54, 1098309})},
		{Ints([]int{}), Ints([]int{})},
	}

	for _, test := range tests {
		InsertionSort(test.In)
		failed := false
		for i, v := range test.In {
			if v != test.Want[i] {
				t.Errorf("InsertionSort, position %v, sortval %v, supposed to be %v", i, v, test.Want[i])
				failed = true
			}
		}
		if failed {
			t.Errorf("Failing Ints: %v != %v", test.In, test.Want)
		}
	}
}
