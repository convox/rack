package various_test

import (
	"github.com/twmb/algoimpl/go/various"
	"testing"
)

func TestSelectOrder(t *testing.T) {
	tests := []struct {
		InI, Want int
		InS       []int
	}{
		{InI: 0, Want: 0, InS: []int{0, 1, 2, 3}},
		{InI: 0, Want: 0, InS: []int{3, 2, 1, 0}},
		{InI: 1, Want: 1, InS: []int{0, 1, 2, 3}},
		{InI: 1, Want: 1, InS: []int{3, 2, 1, 0}},
		{InI: 2, Want: 2, InS: []int{0, 1, 2, 3}},
		{InI: 2, Want: 2, InS: []int{3, 2, 1, 0}},
		{InI: 3, Want: 3, InS: []int{0, 1, 2, 3}},
		{InI: 3, Want: 3, InS: []int{3, 2, 1, 0}},
	}

	for _, test := range tests {
		got := various.SelectOrder(test.InI, test.InS)
		if got != test.Want {
			t.Errorf("Got value %v != expected value %v for order %v of slice %v", got, test.Want, test.InI, test.InS)
		}
	}
}
