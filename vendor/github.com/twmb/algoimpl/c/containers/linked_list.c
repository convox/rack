#include <stdlib.h>

#include "linked_list.h"

linked_list new_linked_list() {
  linked_list l;
  l.head = NULL;
  return l;
}

void delete_ll(linked_list *list) {
  while(list->head != NULL) {
    item *next = list->head->next;
    free(list->head);
    list->head = next;
  }
}

// Adds a new item to the linked list
void push_ll(linked_list *list, void *data) {
  item *new_item = (item *)malloc(sizeof(item));
  if (!new_item) {
    return;
  }
  new_item->data = data;
  new_item->next = list->head;
  list->head = new_item;
}

// Removes and returns pointer to the data at the front of the list
void *pop_ll(linked_list *list) {
  if (list->head == NULL) {
    return NULL;
  }
  void *r_val = list->head->data;
  item *next = list->head->next;
  free(list->head);
  list->head = next;
  return r_val;
}

void *remove_element_ll(linked_list *list, void *data) {
  if (list->head == NULL) {
    return NULL;
  }
  item *trail = list->head;
  item *deleteme = list->head->next;
  while (deleteme != NULL && deleteme->data != data) {
    trail = deleteme;
    deleteme = deleteme->next;
  }
  trail->next = deleteme->next;
  void *r_val = deleteme->data;
  free(deleteme);
  return r_val;
}
