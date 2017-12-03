// Package binary has functions for abusing a binary tree.
package binary

type BinaryTree struct {
	root *node
	size int
}

// Returns a new, empty binary tree.
func New() *BinaryTree {
	return &BinaryTree{size: 0}
}

type node struct {
	parent *node
	left   *node
	right  *node
	value  Comparable
}

// A type that implements the comparable interface can be used in binary trees.
type Comparable interface {
	// Returns -1 if other is less, 0 if they are equal and 1 if other is greater.
	CompareTo(other Comparable) int
}

func walkInOrder(node *node, walked []Comparable) []Comparable {
	if node != nil {
		walked = walkInOrder(node.left, walked)
		walked = append(walked, node.value)
		walked = walkInOrder(node.right, walked)
		return walked
	}
	return nil
}

// Returns an in order Comparable slice of the binary tree.
func (b *BinaryTree) Walk() []Comparable {
	walked := make([]Comparable, 0, b.size)
	return walkInOrder(b.root, walked)
}

func walkPreOrder(node *node, walked []Comparable) []Comparable {
	if node != nil {
		walked = append(walked, node.value)
		walked = walkPreOrder(node.left, walked)
		walked = walkPreOrder(node.right, walked)
		return walked
	}
	return nil
}

// Returns a pre order Comparable slice of the binary tree.
func (b *BinaryTree) WalkPreOrder() []Comparable {
	walked := make([]Comparable, 0, b.size)
	return walkPreOrder(b.root, walked)
}

func walkPostOrder(node *node, walked []Comparable) []Comparable {
	if node != nil {
		walked = walkPostOrder(node.left, walked)
		walked = walkPostOrder(node.right, walked)
		walked = append(walked, node.value)
		return walked
	}
	return nil
}

// Returns a post order Comparable slice of the binary tree.
func (b *BinaryTree) WalkPostOrder() []Comparable {
	walked := make([]Comparable, 0, b.size)
	return walkPostOrder(b.root, walked)
}

// Search for a Comparable in the tree and returns the node that contains it.
// If no node does, returns nil.
func (b *BinaryTree) search(target Comparable) *node {
	current := b.root
	for current != nil {
		switch current.value.CompareTo(target) {
		case -1:
			current = current.left
		case 0:
			return current
		case 1:
			current = current.right
		}
	}
	return nil
}

// Returns true if the binary tree contains the target Comparable
func (b *BinaryTree) Contains(target Comparable) bool {
	return b.search(target) != nil
}

// returns a pointer to the node that contains the minimum
// starting from the start node
func minimum(startNode *node) *node {
	current := startNode
	for current.left != nil {
		current = current.left
	}
	return current
}

// Returns a pointer to a copy of the minimum value
// in the binary tree. If the tree is empty, this will
// return nil.
func (b *BinaryTree) Minimum() *Comparable {
	return &minimum(b.root).value
}

// Returns a pointer to a copy of the maximum value
// in the binary tree. If the tree is empty, this will
// return nil.
func (b *BinaryTree) Maximum() *Comparable {
	current := b.root
	for current.right != nil {
		current = current.right
	}
	// copy
	rval := current.value
	return &rval
}

// Returns a pointer to the successor value of a target Comparable in a binary tree.
// The pointer will be nil if the tree does not contain the target
// of if there is no successor (i.e., you want the successor to the maximum value)
func (b *BinaryTree) Successor(target Comparable) *Comparable {
	current := b.search(target)
	if current == nil {
		return nil
	}
	if current.right != nil {
		return &minimum(current.right).value
	}
	if current == b.root { // if root has no right side, at max, no successor
		return nil
	}
	parent := current.parent
	for parent != nil && current == parent.right {
		current = parent
		parent = current.parent
	}
	return &current.value
}

// Inserts a comparable value into a binary tree.
// Expected running time O(lg n), worst case running time O(n)
// for a tree with n nodes.
func (b *BinaryTree) Insert(value Comparable) {
	newNode := new(node)
	newNode.value = value
	y := (*node)(nil)
	x := b.root
	for x != nil {
		y = x
		if value.CompareTo(x.value) == -1 { // less
			x = x.left
		} else {
			x = x.right
		}
	}
	newNode.parent = y
	if newNode.parent == nil {
		b.root = newNode
	} else if newNode.value.CompareTo(y.value) == -1 {
		y.left = newNode
	} else {
		y.right = newNode
	}
	b.size++
}

// Sets the replacement node's parent information to that
// of the old node and updates the parent of the old node
// to point to the replacement. This also updates pointer
// values on the old node to nil. For garbage collection.
func (b *BinaryTree) transplant(old, replacement *node) {
	if old.parent == nil {
		b.root = replacement
	} else if old == old.parent.left {
		old.parent.left = replacement
	} else {
		old.parent.right = replacement
	}
	if replacement != nil {
		replacement.parent = old.parent
	}
	// garbage collect
	old.parent = nil
	old.left = nil
	old.right = nil
}

// Deletes and returns a pointer to a Comparable value from
// the binary tree. If the value was not in the binary tree,
// this function returns nil. Running time is O(h) for a tree of
// height h.
func (b *BinaryTree) Delete(value Comparable) *Comparable {
	// first find the node because of my decision to not allow access to nodes.
	node := b.search(value)
	if node == nil {
		return nil
	}
	if node.left == nil {
		b.transplant(node, node.right)
	} else if node.right == nil {
		b.transplant(node, node.left)
	} else {
		replacement := minimum(node.right)
		if replacement.parent != node {
			b.transplant(replacement, replacement.right) // sever ties
			replacement.right = node.right
		}
		replacement.right.parent = replacement
		replacement.left = node.left
		replacement.left.parent = replacement
		b.transplant(node, replacement)
	}
	return &node.value
}
