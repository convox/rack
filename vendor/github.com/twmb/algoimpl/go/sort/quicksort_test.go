package sort

import "testing"

var ints = []int{74, 59, 238, -784, 9845, 959, 905, 0, 0, 42, 7586, -5467984, 7586}

func TestQuickSort(t *testing.T) {
	// 100% line coverage
	tests := []struct {
		In, Want Ints
	}{
		{Ints([]int{21, -10, 54, 0, 1098309}), Ints([]int{-10, 0, 21, 54, 1098309})},
		{Ints([]int{}), Ints([]int{})},
		{Ints([]int{74, 59, 238, -784, 9845, 959, 905, 0, 0, 42, 7586, -5467984, 7586}), Ints([]int{-5467984, -784, 0, 0, 42, 59, 74, 238, 905, 959, 7586, 7586, 9845})},
	}

	for _, test := range tests {
		QuickSort(test.In)
		failed := false
		for i, v := range test.In {
			if v != test.Want[i] {
				t.Errorf("QuickSort, position %v, sortval %v, supposed to be %v", i, v, test.Want[i])
				failed = true
			}
		}
		if failed {
			t.Errorf("Failing Ints: %v != %v", test.In, test.Want)
		}
	}
}
