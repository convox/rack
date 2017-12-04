package integers

import "testing"

func TestCountingSort(t *testing.T) {
	tests := []struct {
		In, Want       []int
		InMinV, InMaxV int
	}{
		{[]int{21, -10, 21, 0, 22}, []int{-10, 0, 21, 21, 22}, -10, 22},
		{[]int{}, []int{}, 0, 0},
		{[]int{4, 3, 2, 1, 2, 0, 0, 0, 0, 1, 3, 4, 2, 3, 4, 3}, []int{0, 0, 0, 0, 1, 1, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4}, 0, 4},
	}

	for _, test := range tests {
		got := CountingSort(test.In, test.InMinV, test.InMaxV)
		failed := false
		for i, v := range got {
			if v != test.Want[i] {
				t.Errorf("CountingSort, position %v, sortval %v, supposed to be %v", i, v, test.Want[i])
				failed = true
			}
		}
		if failed {
			t.Errorf("Failing Ints: %v != %v", test.In, test.Want)
		}
	}
}
