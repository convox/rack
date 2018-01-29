package binary

import (
	"testing"
)

type Int int

// Returns -1 if other is not a comparable
func (i Int) CompareTo(other Comparable) int {
	o, ok := other.(Int)
	if !ok || i < o {
		return -1
	} else if i == o {
		return 0
	} else {
		return 1
	}
}

func TestNew(t *testing.T) {
	got := New()
	if got.root != nil || got.size != 0 {
		t.Errorf("New produced incorrect empty binary tree %v", got)
	}
}

func verify(n *node, t *testing.T) {
	if n != nil {
		if n.left != nil {
			compared := n.left.value.CompareTo(n.value)
			if compared == 1 {
				t.Errorf("Left child %v > parent %v", n.left.value, n.value)
			}
		}
		if n.right != nil {
			compared := n.right.value.CompareTo(n.value)
			if compared == -1 {
				t.Errorf("Right child %v < parent %v", n.right.value, n.value)
			}
		}
		verify(n.left, t)
		verify(n.right, t)
	}
}

func TestInsert(t *testing.T) {
	tree := New()
	verify(tree.root, t)
	for i := 0; i < 10; i++ {
		tree.Insert(Int(i))
		verify(tree.root, t)
	}
	tree = New()
	verify(tree.root, t)
	for i := 10; i >= 0; i-- {
		tree.Insert(Int(i))
		verify(tree.root, t)
	}
	tree = New()
	tree.Insert(Int(5)) //         5
	tree.Insert(Int(3)) //   3         8
	tree.Insert(Int(2)) // 2   3    6     9
	tree.Insert(Int(3)) //0     4  5 7
	tree.Insert(Int(0)) // 1
	tree.Insert(Int(1))
	tree.Insert(Int(4))
	tree.Insert(Int(8))
	tree.Insert(Int(6))
	tree.Insert(Int(9))
	tree.Insert(Int(5))
	tree.Insert(Int(7))
	verify(tree.root, t)
}

func TestDelete(t *testing.T) {
	tree := New()
	for i := 0; i < 10; i++ {
		tree.Insert(Int(i))
	}
	for i := 0; i < 10; i++ {
		returned := tree.Delete(Int(i)) // deletes root each time
		if (*returned).(Int) != Int(i) {
			t.Errorf("returned value %v != expected %v", *returned, i)
		}
		verify(tree.root, t)
	}
	tree = New()
	for i := 10; i >= 0; i-- {
		tree.Insert(Int(i))
	}
	for i := 10; i >= 0; i-- {
		returned := tree.Delete(Int(i))
		if (*returned).(Int) != Int(i) {
			t.Errorf("returned value %v != expected %v", *returned, i)
		}
		verify(tree.root, t)
	}
	tree = New()
	tree.Insert(Int(5)) //         5
	tree.Insert(Int(3)) //   3         8
	tree.Insert(Int(2)) // 2   3    6     9
	tree.Insert(Int(3)) //0     4  5 7
	tree.Insert(Int(0)) // 1
	tree.Insert(Int(1))
	tree.Insert(Int(4))
	tree.Insert(Int(8))
	tree.Insert(Int(6))
	tree.Insert(Int(9))
	tree.Insert(Int(5))
	tree.Insert(Int(7))
	tree.Delete(Int(5))
	verify(tree.root, t)
	tree.Delete(Int(3))
	verify(tree.root, t)
	tree.Delete(Int(2))
	verify(tree.root, t)
	tree.Delete(Int(3))
	verify(tree.root, t)
	tree.Delete(Int(0))
	verify(tree.root, t)
	tree.Delete(Int(4))
	verify(tree.root, t)
	tree.Delete(Int(8))
	verify(tree.root, t)
	tree.Delete(Int(6))
	verify(tree.root, t)
	tree.Delete(Int(9))
	verify(tree.root, t)
	tree.Delete(Int(5))
	verify(tree.root, t)
	tree.Delete(Int(1))
	verify(tree.root, t)
	tree.Delete(Int(7))
	verify(tree.root, t)
}

func TestWalk(t *testing.T) {
	tree := New()
	tree.Insert(Int(5)) //         5
	tree.Insert(Int(3)) //   3         8
	tree.Insert(Int(2)) // 2   3    6     9
	tree.Insert(Int(3)) //0     4  5 7
	tree.Insert(Int(0)) // 1
	tree.Insert(Int(1))
	tree.Insert(Int(4))
	tree.Insert(Int(8))
	tree.Insert(Int(6))
	tree.Insert(Int(9))
	tree.Insert(Int(5))
	tree.Insert(Int(7))
	verify(tree.root, t)
	walked := tree.Walk()
	for i := 0; i < len(walked)-1; i++ {
		if walked[i].(Int).CompareTo(walked[i+1]) == 1 {
			t.Errorf("In order walk out of order results: %v before %v", walked[i], walked[i+1])
		}
	}
}

func TestWalkPostOrder(t *testing.T) {
	tree := New()
	for i := 0; i < 10; i++ {
		tree.Insert(Int(i))
	}
	tree.Insert(Int(9))
	verify(tree.root, t)
	walked := tree.Walk()
	for i := 0; i < len(walked)-1; i++ {
		if walked[i].(Int).CompareTo(walked[i+1]) == -1 {
			t.Errorf("Post order walk out of order results: %v after %v", walked[i], walked[i+1])
		}
	}
}

// one long left branch - should walk in ascending order
func TestWalkPreOrder(t *testing.T) {
	tree := New()
	for i := 0; i < 10; i++ {
		tree.Insert(Int(10 - i))
	}
	tree.Insert(Int(1))
	verify(tree.root, t)
	walked := tree.Walk()
	for i := 0; i < len(walked)-1; i++ {
		if walked[i].(Int).CompareTo(walked[i+1]) == 1 {
			t.Errorf("Post order walk out of order results: %v after %v", walked[i], walked[i+1])
		}
	}
}
