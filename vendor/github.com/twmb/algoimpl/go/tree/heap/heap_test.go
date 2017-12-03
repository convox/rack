package heap

// Note: I should add more test cases similar to what is in container/heap/heap.go file,
// ...but this is good.
import (
	"testing"
)

type Ints []int

func (p *Ints) Len() int             { return len(*p) }
func (p *Ints) Less(i, j int) bool   { return (*p)[i] < (*p)[j] }
func (p *Ints) Swap(i, j int)        { (*p)[i], (*p)[j] = (*p)[j], (*p)[i] }
func (p *Ints) Push(v interface{})   { *p = append(*p, v.(int)) }
func (p *Ints) Pop() (v interface{}) { *p, v = (*p)[:p.Len()-1], (*p)[p.Len()-1]; return }

// Untested Init function, but it was tested before I deleted At() - it works, do not want to write right now.
func TestPush(t *testing.T) {
	tests := []struct {
		In       Interface
		PushVal  interface{}
		WantInts Ints
	}{
		{
			Interface((*Ints)(&[]int{})),
			0,
			Ints([]int{0}),
		},
		{
			Interface((*Ints)(&[]int{16, 14, 10, 8, 7, 9, 3, 2, 4, 1})),
			15,
			Ints([]int{16, 15, 10, 8, 14, 9, 3, 2, 4, 1, 7}),
		},
	}
	for _, test := range tests {
		Push(test.In, test.PushVal)
		changedInts := test.In.(*Ints)
		failed := false
		for i, v := range *changedInts {
			if v != test.WantInts[i] {
				failed = true
				break
			}
		}
		if failed {
			t.Errorf("Failing Ints: result %v != want %v", changedInts, test.WantInts)
		}
	}
}

func TestPop(t *testing.T) {
	tests := []struct {
		In       Interface
		PopIndex int
		WantInts Ints
		WantV    int
	}{
		{
			Interface((*Ints)(&[]int{16, 14, 10, 8, 7, 9, 3, 2, 4, 1})),
			1,
			Ints([]int{14, 8, 10, 4, 7, 9, 3, 2, 1}),
			//Ints([]int{16, 8, 10, 4, 7, 9, 3, 2, 1}),
			16,
		},
	}
	for _, test := range tests {
		got := Pop(test.In).(int)
		if test.WantV != got {
			t.Errorf("Return value %v != wanted %v", got, test.WantV)
		}
		changedInts := test.In.(*Ints)
		failed := false
		for i, v := range *changedInts {
			if v != test.WantInts[i] {
				failed = true
				break
			}
		}
		if failed {
			t.Errorf("Failing Ints: result %v != want %v", changedInts, test.WantInts)
		}
	}
}

// Taken from Go source code "heap_test.go" and modified to fit my structures
// I must learn to make tests this easy...
func (h Ints) verify(t *testing.T, i int) {
	n := h.Len()
	left := 2*i + 1
	right := 2*i + 2
	if left < n {
		if h.Less(i, left) {
			t.Errorf("heap invariant invalidated [%d] = %d > [%d] = %d", i, h[i], left, h[left])
			return
		}
		h.verify(t, left)
	}
	if right < n {
		if h.Less(i, right) {
			t.Errorf("heap invariant invalidated [%d] = %d > [%d] = %d", i, h[i], left, h[right])
			return
		}
		h.verify(t, right)
	}
}

func TestRemove1(t *testing.T) {
	h := new(Ints)
	for i := 0; i < 10; i++ {
		Push(h, i)
	}
	h.verify(t, 0)
	// removes the max every time
	for i := 0; h.Len() > 0; i++ {
		x := Remove(h, 0)
		if x.(int) != 9-i {
			t.Errorf("Remove(0) got %d; want %d", x, i)
		}
		h.verify(t, 0)
	}
}

func TestRemove2(t *testing.T) {
	N := 10
	h := new(Ints)
	for i := 0; i < N; i++ {
		Push(h, i)
	}
	h.verify(t, 0)
	// tests that it removed all
	m := make(map[int]bool)
	for h.Len() > 0 {
		x := Remove(h, (h.Len()-1)/2) // remove from middle
		m[x.(int)] = true
		h.verify(t, 0)
	}
	if len(m) != N {
		t.Errorf("len(m) = %d; want %d", len(m), N)
	}
	// and removed all correctly
	for i := 0; i < len(m); i++ {
		if !m[i] {
			t.Errorf("m[%d] doesn't exist", i)
		}
	}
}
