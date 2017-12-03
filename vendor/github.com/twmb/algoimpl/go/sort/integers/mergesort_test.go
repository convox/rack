package integers

import "testing"

func TestMergeSort(t *testing.T) {
	tests := []struct {
		In, Want []int
	}{
		{[]int{21, -10, 54, 0, 1098309}, []int{-10, 0, 21, 54, 1098309}},
		{[]int{}, []int{}},
		{[]int{4, 3, 2, 1, 2, 3}, []int{1, 2, 2, 3, 3, 4}},
	}

	for _, test := range tests {
		tch := make(chan int)
		got := make([]int, 0, len(test.In))
		go MergeSort(test.In, 0, len(test.In), tch)
		for v := range tch {
			got = append(got, v)
		}
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
