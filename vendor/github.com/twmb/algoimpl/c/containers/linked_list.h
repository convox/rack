#ifndef GITHUB_TWMB_C_LL
#define GITHUB_TWMB_C_LL

// Push, remove
typedef struct item {
  struct item *next;
  void *data;
} item;

typedef struct {
  item *head;
} linked_list;

linked_list new_linked_list();
void delete_ll(linked_list *list);
void push_ll(linked_list *list, void *data);
void *pop_ll(linked_list *list);
void *remove_element_ll(linked_list *list, void *data);

#endif
