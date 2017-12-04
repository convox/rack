#include <stdio.h>
#include <stdint.h>

#include "linked_list.h"

int main() {
  linked_list mine = new_linked_list();
  int failed = 0;
  push_ll(&mine, (void *)3);
  push_ll(&mine, (void *)2);
  push_ll(&mine, (void *)2);
  remove_element_ll(&mine, (void *)2);
  delete_ll(&mine);
  if (mine.head != NULL) {
    printf("linked list head not null after trying to delete all elements\n");
    failed = -1;
  }
  return failed;
}
