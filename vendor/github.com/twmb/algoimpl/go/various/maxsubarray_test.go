package various

import "testing"

var tests = []struct {
	In                      []int
	WantLI, WantRI, WantSum int
}{
	{[]int{}, 0, 0, 0},
	{[]int{-1}, 0, 1, -1},
	{[]int{3, -1, -1, 4}, 0, 4, 5},        //whole thing
	{[]int{-1, 1, 1, -1}, 1, 3, 2},        //crossing
	{[]int{-1, -2, 1, 2}, 2, 4, 3},        //right side
	{[]int{1, 2, -3, -4}, 0, 2, 3},        //left side
	{[]int{1, -2, -3, 5, 6, 7}, 3, 6, 18}, // 6 length, right side
	{[]int{1, -2, -3, 5, 6}, 3, 5, 11},    // 5 length, right side
}

func TestMaxSubarrayRecursive(t *testing.T) {
	for _, test := range tests {
		li, ri, sum := MaxSubarrayRecursive(test.In, 0, len(test.In))
		if li != test.WantLI || ri != test.WantRI || sum != test.WantSum {
			t.Errorf("test results (%v, %v, %v) != test expected (%v, %v, %v) for array %v", li, ri, sum, test.WantLI, test.WantRI, test.WantSum, test.In)
		}
	}
	li, ri, sum := MaxSubarrayRecursive(tests[2].In, 2, len(tests[2].In)) // test half
	if li != 3 || ri != 4 || sum != 4 {
		t.Errorf("test results (%v, %v, %v) != test expected (%v, %v, %v) for right half test of array %v", li, ri, sum, 3, 4, 4, tests[2].In)
	}
	li, ri, sum = MaxSubarrayRecursive(tests[3].In, 0, 2) // test left
	if li != 1 || ri != 2 || sum != 1 {
		t.Errorf("test results (%v, %v, %v) != test expected (%v, %v, %v) for right half test of array %v", li, ri, sum, 1, 2, 1, tests[3].In)
	}
	li, ri, sum = MaxSubarrayRecursive(tests[5].In, 1, 3) // test middle
	if li != 1 || ri != 2 || sum != 2 {
		t.Errorf("test results (%v, %v, %v) != test expected (%v, %v, %v) for right half test of array %v", li, ri, sum, 1, 2, 2, tests[5].In)
	}

}

func TestMaxSubarray(t *testing.T) {
	for _, test := range tests {
		li, ri, sum := MaxSubarray(test.In, 0, len(test.In))
		if li != test.WantLI || ri != test.WantRI || sum != test.WantSum {
			t.Errorf("test results (%v, %v, %v) != test expected (%v, %v, %v) for array %v", li, ri, sum, test.WantLI, test.WantRI, test.WantSum, test.In)
		}
	}
	li, ri, sum := MaxSubarray(tests[2].In, 2, len(tests[2].In)) // test half
	if li != 3 || ri != 4 || sum != 4 {
		t.Errorf("test results (%v, %v, %v) != test expected (%v, %v, %v) for right half test of array %v", li, ri, sum, 3, 4, 4, tests[2].In)
	}
	li, ri, sum = MaxSubarray(tests[3].In, 0, 2) // test left
	if li != 1 || ri != 2 || sum != 1 {
		t.Errorf("test results (%v, %v, %v) != test expected (%v, %v, %v) for right half test of array %v", li, ri, sum, 1, 2, 1, tests[3].In)
	}
	li, ri, sum = MaxSubarray(tests[5].In, 1, 3) // test middle
	if li != 1 || ri != 2 || sum != 2 {
		t.Errorf("test results (%v, %v, %v) != test expected (%v, %v, %v) for right half test of array %v", li, ri, sum, 1, 2, 2, tests[5].In)
	}
}

var tests2 = []struct {
	In, Want []int
	WantSum  int
}{
	{[]int{}, []int{}, 0},
	{[]int{-1}, []int{-1}, -1},
	{[]int{3, -1, -1, 4}, []int{3, -1, -1, 4}, 5},   //whole thing
	{[]int{-1, 1, 1, -1}, []int{1, 1}, 2},           //crossing
	{[]int{-1, -2, 1, 2}, []int{1, 2}, 3},           //right side
	{[]int{1, 2, -3, -4}, []int{1, 2}, 3},           //left side
	{[]int{1, -2, -3, 5, 6, 7}, []int{5, 6, 7}, 18}, // 6 length, right side
	{[]int{1, -2, -3, 5, 6}, []int{5, 6}, 11},       // 5 length, right side
}

func TestMaxSubarray2(t *testing.T) {
	for testI, test := range tests2 {
		got, sum := MaxSubarray2(test.In)
		if len(got) != len(test.Want) {
			t.Errorf("test %v: got slice len != want slice len, got: %v, want: %v", testI, len(got), len(test.Want))
			continue
		}
		for i := 0; i < len(got); i++ {
			if got[i] != test.Want[i] {
				t.Errorf("wanted %v at index %v, got %v", test.Want[i], i, got[i])
			}
		}
		if sum != test.WantSum {
			t.Errorf("test %v: got sum %v != want sum %v", testI, sum, test.WantSum)
		}
	}
}
