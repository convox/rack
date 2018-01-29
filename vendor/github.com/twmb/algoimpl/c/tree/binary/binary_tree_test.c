#include <stdio.h>
#include <stdint.h>
#include <stdbool.h>

#include "binary_tree.h"

bool less(void *left_data, void *right_data) {
  if ((int64_t)left_data < (int64_t)right_data) {
    return true;
  }
  return false;
}

void print_node(void *data) {
  printf("%lu ", (int64_t)data);
}

int main() {
  // dirty repetitive tests, but thorough
  int failed = 0;
  struct binary_tree tree = new_binary_tree(); // tree of ints
  push_binary_tree(&tree, (void *)1, &less);
  push_binary_tree(&tree, (void *)0, &less);
  push_binary_tree(&tree, (void *)1, &less);
  push_binary_tree(&tree, (void *)3, &less); //       1  
  push_binary_tree(&tree, (void *)4, &less); //    0     1  
  push_binary_tree(&tree, (void *)2, &less); //              3       
  push_binary_tree(&tree, (void *)1, &less); //          2       4      
  push_binary_tree(&tree, (void *)2, &less); //        1   2   3   9    
  push_binary_tree(&tree, (void *)3, &less); //                   8 11
  push_binary_tree(&tree, (void *)9, &less); //             
  push_binary_tree(&tree, (void *)11, &less);
  push_binary_tree(&tree, (void *)8, &less);
  void *returned;
  returned = delete_node_binary_tree(&tree.root); // 1
  if ((int64_t) returned != 1) {
    printf("deleted node value %lu != expected %d\n", (int64_t)returned, 1);
    failed = -1;
  }
  returned = delete_node_binary_tree(&tree.root); // 1
  if ((int64_t) returned != 1) {
    printf("deleted node value %lu != expected %d\n", (int64_t)returned, 1);
    failed = -1;
  }
  returned = delete_node_binary_tree(&tree.root); // 1
  if ((int64_t) returned != 1) {
    printf("deleted node value %lu != expected %d\n", (int64_t)returned, 1);
    failed = -1;
  }
  returned = delete_node_binary_tree(&tree.root); // 2
  if ((int64_t) returned != 2) {
    printf("deleted node value %lu != expected %d\n", (int64_t)returned, 2);
    failed = -1;
  }
  returned = delete_node_binary_tree(&tree.root); // 2
  if ((int64_t) returned != 2) {
    printf("deleted node value %lu != expected %d\n", (int64_t)returned, 2);
    failed = -1;
  }
  returned = delete_node_binary_tree(&tree.root); // 3
  if ((int64_t) returned != 3) {
    printf("deleted node value %lu != expected %d\n", (int64_t)returned, 3);
    failed = -1;
  }
  returned = delete_node_binary_tree(&tree.root); // 3
  if ((int64_t) returned != 3) {
    printf("deleted node value %lu != expected %d\n", (int64_t)returned, 3);
    failed = -1;
  }
  returned = delete_node_binary_tree(&tree.root); // 4
  if ((int64_t) returned != 4) {
    printf("deleted node value %lu != expected %d\n", (int64_t)returned, 4);
    failed = -1;
  }
  returned = delete_node_binary_tree(&tree.root); // 8 
  if ((int64_t) returned != 8) {
    printf("deleted node value %lu != expected %d\n", (int64_t)returned, 8);
    failed = -1;
  }
  returned = delete_node_binary_tree(&tree.root); // 9 
  if ((int64_t) returned != 9) {
    printf("deleted node value %lu != expected %d\n", (int64_t)returned, 9);
    failed = -1;
  }
  returned = delete_node_binary_tree(&tree.root); // 11 
  if ((int64_t) returned != 11) {
    printf("deleted node value %lu != expected %d\n", (int64_t)returned, 11);
    failed = -1;
  }
  returned = delete_node_binary_tree(&tree.root); // 0
  if ((int64_t) returned != 0) {
    printf("deleted node value %lu != expected %d\n", (int64_t)returned, 0);
    failed = -1;
  }
  return failed;
}

