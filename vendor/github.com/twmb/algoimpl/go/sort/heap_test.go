package sort

import "testing"

// uses Ints from sort_test

func TestHeapSort(t *testing.T) {
	tests := []struct {
		In, Want Ints
	}{
		{Ints([]int{}), Ints([]int{})},
		{
			Ints([]int{4, 1, 3, 2, 16, 9, 10, 14, 8, 7}),
			Ints([]int{1, 2, 3, 4, 7, 8, 9, 10, 14, 16}),
		},
	}
	for _, test := range tests {
		HeapSort(test.In)
		failed := false
		for i, v := range test.In {
			if v != test.Want[i] {
				failed = true
				break
			}
		}
		if failed {
			t.Errorf("Failing Ints: result %v != want %v", test.In, test.Want)
		}
	}
}
