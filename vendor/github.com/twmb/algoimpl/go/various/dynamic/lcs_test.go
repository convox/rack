package dynamic_test

import (
	"github.com/twmb/algoimpl/go/various/dynamic"
	"testing"
)

func TestLongestCommonSubsequence(t *testing.T) {
	tests := []struct {
		InF, InS, Want string
	}{
		{"hi", "hello", "h"},
		{"ABCBDAB", "BDCABA", "BCBA"},
		{"thisisatest", "testing123testing", "tsitest"},
		{"1234", "1224533324", "1234"},
		{"", "", ""},
	}

	for _, test := range tests {
		lcs := dynamic.LongestCommonSubsequence(test.InF, test.InS)
		if lcs != test.Want {
			t.Errorf("Input %v, %v recieved output %v, != want %v", test.InF, test.InS, lcs, test.Want)
		}
	}
}
