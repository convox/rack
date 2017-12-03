#include <stdio.h>
#include <stdint.h>

#include "heap.c"
#include "dynamic_array.h"

// ******* functions for int[5] array  ********
bool ints_less(void *c, int l, int r) {
  int *a = c;
  return a[l] < a[r];
}
int ints_len(void *container) {
  return 15;
}
void ints_swap(void *container, int left, int right) {
  if (left != right) {
    int *d = container;
    d[left] ^= d[right];
    d[right] ^= d[left];
    d[left] ^= d[right];
  }
}
// ******* functions for dynamic array ********
bool dynarr_less(void *container, int left, int right) {
  dynarr **d = container;
  return dynarr_at(*d, left) < dynarr_at(*d, right);
}
int dynarr_lenfunc(void *container) {
  return dynarr_len(*(dynarr **)container);
}
void dynarr_swap(void *container, int left, int right) {
  if (left != right) {
    dynarr **d = container;
    dynarr_set(*d, left, (void *)((int64_t)dynarr_at(*d, left) ^ (int64_t)dynarr_at(*d, right)));
    dynarr_set(*d, right,(void *)((int64_t)dynarr_at(*d, left) ^ (int64_t)dynarr_at(*d, right)));
    dynarr_set(*d, left, (void *)((int64_t)dynarr_at(*d, left) ^ (int64_t)dynarr_at(*d, right)));
  }
}
void dynarr_push(void *d, void *elem) {
  dynarr_append(*(dynarr **)d, elem);
}
void *dynarr_pop(void *container) {
  dynarr **d = container;
  void *end = dynarr_at(*d, dynarr_len(*d) - 1);
  dynarr *n = dynarr_slice(*d, 0, dynarr_len(*d) - 1);
  destroy_dynarr(*d);
  *d = n;
  return end;
}
// ******* ********* *** ******* ***** ********

Heap myheap;

struct indexcalc {
  int in, want;
};

int check_valid_intheap(int *array, int current, int len) {
  int rval = 0;
  int lc = lchild(current);
  int rc = rchild(current);

  if (lc < len) {
    if (array[lc] < array[current]) {
      printf("array left child %d < parent %d\n", array[lc], array[current]);
      printf("[");
      for (int i = 0; i < len; i++) {
        printf("%d ", array[i]);
      }
      printf("]\n");
      return -1;
    }
    rval |= check_valid_intheap(array, lc, len);
  }
  if (rc < len) {
    if (array[rc] < array[current]) {
      printf("array right child %d < parent %d\n", array[rc], array[current]);
      printf("[");
      for (int i = 0; i < len; i++) {
        printf("%d ", array[i]);
      }
      printf("]\n");
      return -1;
    }
    rval |= check_valid_intheap(array, rc, len);
  }
  return rval;
}

int check_valid_dynarr_heap(dynarr *d, int current, int len) {
  int rval = 0;
  int lc = lchild(current);
  int rc = rchild(current);

  if (lc < len) {
    if (dynarr_at(d, lc) < dynarr_at(d, current)) {
      printf("dynarr left child %lu < parent %lu\n", (int64_t)dynarr_at(d, lc), (int64_t)dynarr_at(d, current));
      printf("[");
      for (int i = 0; i < len; i++) {
        printf("%lu ", (int64_t)dynarr_at(d, i));
      }
      printf("]\n");
      return -1;
    }
    rval |= check_valid_dynarr_heap(d, lc, len);
  }
  if (rc < len) {
    if (dynarr_at(d, rc) < dynarr_at(d, current)) {
      printf("dynarr right child %lu < parent %lu\n", (int64_t)dynarr_at(d, rc), (int64_t)dynarr_at(d, current));
      printf("[");
      for (int i = 0; i < len; i++) {
        printf("%lu ", (int64_t)dynarr_at(d, i));
      }
      printf("]\n");
      return -1;
    }
    rval |= check_valid_dynarr_heap(d, rc, len);
  }
  return rval;
}

int test_parent(void) {
  struct indexcalc tests[] = {
    {.in = 0, .want = 0},
    {.in = 1, .want = 0}, // lchild
    {.in = 2, .want = 0}, // rchild
    {.in = 7, .want = 3}, // lchild
    {.in = 8, .want = 3}, // rchild
  };

  for (int i = 0; i < sizeof(tests)/sizeof(struct indexcalc); i++) {
    if (parent(tests[i].in) != tests[i].want) {
      printf("heapstatic: parent failed for input %d, returned %d, expected %d\n", tests[i].in, parent(tests[i].in), tests[i].want);
      return -1;
    }
  }
  return 0;
}

int test_lchild(void) {
  struct indexcalc tests[] = {
    {.in = 0, .want = 1},
    {.in = 3, .want = 7}, // lchild
  };

  for (int i = 0; i < sizeof(tests)/sizeof(struct indexcalc); i++) {
    if (lchild(tests[i].in) != tests[i].want) {
      printf("heapstatic: lchild failed for input %d, returned %d, expected %d\n", tests[i].in, lchild(tests[i].in), tests[i].want);
      return -1;
    }
  }
  return 0;
}

int test_rchild(void) {
  struct indexcalc tests[] = {
    {.in = 0, .want = 2},
    {.in = 3, .want = 8}, // rchild
  };

  for (int i = 0; i < sizeof(tests)/sizeof(struct indexcalc); i++) {
    if (rchild(tests[i].in) != tests[i].want) {
      printf("heapstatic: rchild failed for input %d, returned %d, expected %d\n", tests[i].in, rchild(tests[i].in), tests[i].want);
      return -1;
    }
  }
  return 0;
}

int test_shuffleUp(void) {
  int rval = 0;

  int container1[5] = { 1, 2, 3, 4, 0 };
  int container2[5] = { 4, 3, 2, 1, 0 };
  int container3[5] = { 0, 1, 2, 3, 4 };

  set_heap_container(myheap, &container1);
  shuffleUp(myheap, 4);
  rval |= check_valid_intheap(container1, 0, 5);

  set_heap_container(myheap, &container2);
  for (int i = 0; i < 5; i++) {
    shuffleUp(myheap, i);
  }
  rval |= check_valid_intheap(container2, 0, 5);

  set_heap_container(myheap, &container3);
  for (int i = 0; i < 5; i++) {
    shuffleUp(myheap, i);
  }
  rval |= check_valid_intheap(container3, 0, 5);

  return rval;
}

int test_shuffleDown(void) {
  int rval = 0;

  int container4[5] = { 4, 0, 1, 2, 3 };
  int container5[5] = { 0, 4, 1, 2, 3 };
  int container6[5] = { 0, 1, 2, 3, 4 };

  set_heap_container(myheap, &container4);
  shuffleDown(myheap, 0, 5);
  rval |= check_valid_intheap(container4, 0, 5);

  set_heap_container(myheap, &container5);
  shuffleDown(myheap, 1, 5);
  rval |= check_valid_intheap(container5, 0, 5);

  set_heap_container(myheap, &container6);
  for (int i = 0; i < 5; i++) {
    shuffleDown(myheap, i, 5);
  }
  rval |= check_valid_intheap(container6, 0, 5);

  return rval;
}

int test_heap_push(void) {
  dynarr *d = create_dynarr();
  set_heap_container(myheap, &d);
  for (int64_t i = 0; i < 20; i++) {
    heap_push(myheap, (void *)i);
  }
  int rval = check_valid_dynarr_heap(d, 0, dynarr_len(d));
  destroy_dynarr(d);
  return rval;
}

int test_heap_pop(void) {
  int rval = 0;
  dynarr *d = create_dynarr();
  set_heap_container(myheap, &d);
  for (int64_t i = 0; i < 20; i++) {
    heap_push(myheap, (void *)i);
  }
  for (int64_t i = 0; i < 20; i++) {
    int64_t r = (int64_t)heap_pop(myheap);
    rval |= check_valid_dynarr_heap(d, 0, dynarr_len(d));
    if (r != i) {
      printf("heap_pop: expected %lu, got %lu\n", i, r);
      rval |= -1;
    }
  }
  destroy_dynarr(d);
  return rval;
}

int test_heap_delete(void) {
  int rval = 0;
  dynarr *d = create_dynarr();
  set_heap_container(myheap, &d);
  for (int64_t i = 0; i < 20; i++) {
    heap_push(myheap, (void *)i);
  }
  for (int64_t i = 0; i < 20; i++) {
    int64_t r = (int64_t)heap_delete(myheap, 0);
    rval |= check_valid_dynarr_heap(d, 0, dynarr_len(d));
    if (r != i) {
      printf("heap_pop: expected %lu, got %lu\n", i, r);
      rval = -1;
    }
  }
  destroy_dynarr(d);
  return rval;
}

int test_heapify(void) {
  int rval = 0;
  dynarr *d = create_dynarr();
  set_heap_container(myheap, &d);
  for (int64_t i = 0; i < 20; i++) {
    dynarr_append(d, (void *)i);
  }
  heapify(myheap);
  rval |= check_valid_dynarr_heap(d, 0, dynarr_len(d));
  destroy_dynarr(d);
  return rval;
}


int main(void) {
  int rval = 0;
  rval |= test_parent();
  rval |= test_lchild();
  rval |= test_rchild();
  // testing shuffles with a int[5], only len, less and swap called
  myheap = create_heap();
  set_heap_lessfunc(myheap, &ints_less);
  set_heap_lenfunc(myheap, &ints_len);
  set_heap_swapfunc(myheap, &ints_swap);

  rval |= test_shuffleUp();
  rval |= test_shuffleDown();

  set_heap_lessfunc(myheap, &dynarr_less);
  set_heap_lenfunc(myheap, &dynarr_lenfunc);
  set_heap_swapfunc(myheap, &dynarr_swap);
  set_heap_pushfunc(myheap, &dynarr_push);
  set_heap_popfunc(myheap, &dynarr_pop);

  rval |= test_heap_push();
  rval |= test_heap_pop();
  rval |= test_heap_delete();

  rval |= test_heapify();

  destroy_heap(myheap);

  return rval;
}
