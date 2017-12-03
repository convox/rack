#include <stdlib.h>
#include <stdio.h>
#include <stdint.h>

#include "binary_tree.h"

struct binary_tree new_binary_tree() {
  struct binary_tree new;
  new.root = NULL;
  return new;
}

int push_binary_tree(struct binary_tree *tree, void *data,
    bool (*less)(void *left_data, void *right_data)) {
  struct node *currentParent = tree->root;
  struct node *current = tree->root;
  while (current != NULL) {
    if (less(data, current->data)) {
      currentParent = current;
      current = current->lchild;
    } else {
      currentParent = current;
      current = current->rchild;
    }
  }
  struct node *new_node = (struct node *)malloc(sizeof(struct node));
  if (!new_node) {
    return -1;
  }
  new_node->parent = currentParent;
  new_node->data = data;
  new_node->lchild = NULL;
  new_node->rchild = NULL;
  if (currentParent == NULL) { // no root element
    tree->root = new_node;
    return 0;
  }
  if (less(new_node->data, currentParent->data)) {
    currentParent->lchild = new_node;
  } else {
    currentParent->rchild = new_node;
  }
  return 0;
}

void transplant(struct node *replace, struct node *replacement) {
  if (replace->parent == NULL) {
    // do nothing
  } else if (replace == replace->parent->lchild) {
    replace->parent->lchild = replacement;
  } else {
    replace->parent->rchild = replacement;
  }
  if (replacement != NULL) {
    replacement->parent = replace->parent;
  }
  replace->parent = NULL;
  replace->lchild = NULL;
  replace->rchild = NULL;
  return;
}

struct node *minimum(struct node *node) {
  if (node == NULL) {
    return NULL;
  }
  struct node *minimum = node;
  while (minimum->lchild != NULL) {
    minimum = minimum->lchild;
  }
  return minimum;
}

struct node *successor(struct node *node) {
  if (node == NULL) {
    return NULL;
  }
  if (node->rchild != NULL) {
    return minimum(node->rchild);
  }
  return node->parent; // may be null
}

void *delete_node_binary_tree(struct node **p_deletenode) {
  if (p_deletenode == NULL || *p_deletenode == NULL) {
    return NULL;
  }
  struct node *deletenode = *p_deletenode;
  void *data = deletenode->data;
  if (deletenode->lchild == NULL) {
    *p_deletenode = deletenode->rchild;
    transplant(deletenode, deletenode->rchild);
    free(deletenode);
    return data;
  } else if (deletenode->rchild == NULL) {
    *p_deletenode = deletenode->lchild;
    transplant(deletenode, deletenode->lchild);
    free(deletenode);
    return data;
  } else {
    // left and right are not nil
    // find successor in right child to replace here
    struct node *suc_node = successor(deletenode); // will not be null
    if (suc_node != deletenode->rchild) {
      transplant(suc_node, suc_node->rchild);
      deletenode->rchild->parent = suc_node;
      suc_node->rchild = deletenode->rchild;
    }
    suc_node->lchild = deletenode->lchild;
    deletenode->lchild->parent = suc_node;
    transplant(deletenode, suc_node);
    free(deletenode);
    *p_deletenode = suc_node;
    return data;
  }
}

void walk_node(struct node *node, void (*print_node)(void *data)) {
  if (node != NULL) {
    walk_node(node->lchild, print_node);
    print_node(node->data);
    walk_node(node->rchild, print_node);
  }
}

