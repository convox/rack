package various_test

import (
	"github.com/twmb/algoimpl/go/various"
	"testing"
)

func TestInversions(t *testing.T) {
	tests := []struct {
		In   []int
		Want int
	}{
		{[]int{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}, 45},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 0},
		{[]int{1, 3, 5, 2, 4, 6}, 3},
		{[]int{}, 0},
	}
	for _, test := range tests {
		count := various.Inversions(test.In)
		if count != test.Want {
			t.Errorf("Input array %v, expected count %v, got %v", test.In, test.Want, count)
		}
	}
}

func BenchmarkInversions5(b *testing.B) {
	b.StopTimer()
	inverse := make([]int, 1<<5)
	for i := 0; i < 1<<5; i++ {
		inverse[i] = 1<<5 - i
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		various.Inversions(inverse)
	}
}

func BenchmarkInversions10(b *testing.B) {
	b.StopTimer()
	inverse := make([]int, 1<<10)
	for i := 0; i < 1<<10; i++ {
		inverse[i] = 1<<10 - i
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		various.Inversions(inverse)
	}
}

func BenchmarkInversions20(b *testing.B) {
	b.StopTimer()
	inverse := make([]int, 1<<20)
	for i := 0; i < 1<<20; i++ {
		inverse[i] = 1<<20 - i
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		various.Inversions(inverse)
	}
}
