#ifndef TWMB_BIN_TREE
#define TWMB_BIN_TREE

#include <stdbool.h>
#include <stdio.h>

struct node {
  struct node *lchild;
  struct node *rchild;
  struct node *parent;
  void *data;
};

struct binary_tree {
  struct node *root;
};

// Returns a new binary tree.
struct binary_tree new_binary_tree();
// Adds data to a binary tree using the passed in less function.
int push_binary_tree(struct binary_tree *tree, void *data,
    bool (*less)(void *left, void *right));
// Walks the binary tree under the input node in order, calling
// the print function for each node's data element.
void walk_node(struct node *node, void (*print_node)(void *data));
// Deletes a node from a binary tree and returns a void * to the
// data that was attached to it.
void *delete_node_binary_tree(struct node **deletenode);
// Finds the minimum node in the tree starting from the input node.
struct node *minimum(struct node *node);

#endif
