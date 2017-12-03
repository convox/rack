package matrix

import (
	"errors"
	"testing"
)

type want struct {
	r [][]int
	e error
}

type test struct {
	inA  [][]int
	inB  [][]int
	want want
}

var tests []test

func init() {
	var test1 test
	var want1 want
	test1.inA = [][]int{[]int{}} // test A empty
	test1.inB = [][]int{
		[]int{1, 0, 0, 0},
	}
	want1.r = nil
	want1.e = errors.New("Cannot multiply empty matrices")
	test1.want = want1

	var test2 test
	var want2 want
	test2.inA = [][]int{
		[]int{1, 0, 0, 0},
	}
	test2.inB = [][]int{[]int{}} // test B empty
	want2.r = nil
	want2.e = errors.New("Cannot multiply empty matrices")
	test2.want = want2

	var test3 test
	var want3 want
	test3.inA = [][]int{
		[]int{1, 0, 0, 0},
	}
	test3.inB = [][]int{[]int{1, 2, 3}} // test dimension mismatch
	want3.r = nil
	want3.e = errors.New("Dimension mismatch")
	test3.want = want3

	var test4 test
	var want4 want
	test4.inA = [][]int{
		[]int{1, 2, 3, 4},
		[]int{1, 2, 3, 4},
	}
	test4.inB = [][]int{
		[]int{1, 1},
		[]int{2, 2},
		[]int{3, 3},
		[]int{4, 4},
	}
	want4.r = [][]int{
		[]int{30, 30},
		[]int{30, 30},
	}
	want4.e = nil
	test4.want = want4

	tests = append(tests, test1)
	tests = append(tests, test2)
	tests = append(tests, test3)
	tests = append(tests, test4)

}

func TestBasicMultiply(t *testing.T) {
	for _, test := range tests {
		result, err := BasicMultiply(test.inA, test.inB)
		if err != nil { // if received error
			if result != nil {
				t.Errorf("wanted result nil for test inA %v, inB %v", test.inA, test.inB)
				continue
			}
			if test.want.e.Error() != err.Error() {
				t.Errorf("BasicMultiply err %v != want err %v", err, test.want.e)
				continue
			}
		}
		// if didn't receive error
		if len(result) != len(test.want.r) {
			t.Errorf("Row count not the same for test %v and result %v", test.want.r, result)
			continue
		} // these two errors should never happen, as BasicMultiply would panic first.
		for r := range test.want.r {
			if len(result[r]) != len(test.want.r[r]) {
				t.Errorf("Column count not the same for row %v of test %v and result %v", r, test.want.r, result)
				continue
			}
			for c := range test.want.r[r] {
				if result[r][c] != test.want.r[r][c] {
					t.Errorf("Values at %v,%v not equal for test %v, result %v", r, c, test.want.r, result)
				}
			}
		}
	}
}

func TestRecursiveMultiply(t *testing.T) {
	for _, test := range tests {
		result, err := RecursiveMultiply(test.inA, test.inB)
		if err != nil { // if received error
			if result != nil {
				t.Errorf("wanted result nil for test inA %v, inB %v", test.inA, test.inB)
				continue
			}
			if test.want.e.Error() != err.Error() {
				t.Errorf("BasicMultiply err %v != want err %v", err, test.want.e)
				continue
			}
		}
		// if didn't receive error
		if len(result) != len(test.want.r) {
			t.Errorf("Row count not the same for test %v and result %v", test.want.r, result)
			continue
		} // these two errors should never happen, as BasicMultiply would panic first.
		for r := range test.want.r {
			if len(result[r]) != len(test.want.r[r]) {
				t.Errorf("Column count not the same for row %v of test %v and result %v", r, test.want.r, result)
				continue
			}
			for c := range test.want.r[r] {
				if result[r][c] != test.want.r[r][c] {
					t.Errorf("Values at %v,%v not equal for test %v, result %v", r, c, test.want.r, result)
				}
			}
		}
	}
}
