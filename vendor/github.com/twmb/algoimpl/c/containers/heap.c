#include <stdlib.h> 

#include "heap.h"

struct heap {
  void   *container;
  bool  (*less)(void *c, int l, int r);
  int   (*len)(void *c);
  void  (*swap)(void *c, int l, int r);
  void  (*push)(void *c, void *e);
  void *(*pop)(void *c);
};

Heap create_heap(void) {
  Heap newheap = malloc(sizeof(struct heap));
  newheap->container = NULL;
  newheap->less      = NULL;
  newheap->len       = NULL;
  newheap->swap      = NULL;
  newheap->push      = NULL;
  newheap->pop       = NULL;
  return newheap;
}

void destroy_heap(Heap h) {
  free(h);
}

void set_heap_container(Heap h, void *container) {
  h->container = container;
}
void set_heap_lessfunc(Heap h, bool (*less)(void*, int, int)) {
  h->less = less;
}
void set_heap_lenfunc(Heap h, int (*len)(void*)) {
  h->len = len;
}
void set_heap_swapfunc(Heap h, void (*swap)(void*, int, int)) {
  h->swap = swap;
}
void set_heap_pushfunc(Heap h, void (*push)(void*, void*)) {
  h->push = push;
}
void set_heap_popfunc(Heap h, void *(*pop)(void*)) {
  h->pop = pop;
}

static inline int parent(int i) {
  return (i - 1) / 2;
}
static inline int lchild(int i) {
  return i * 2 + 1;
}
static inline int rchild(int i) {
  return i * 2 + 2;
}

static void shuffleUp(Heap h, int elem) {
  while (true) {
    int p = parent(elem);
    if (p == elem || !h->less(h->container, elem, p)) {
      break;
    }
    h->swap(h->container, p, elem);
    elem = p;
  }
}

static void shuffleDown(Heap h, int elem, int len) {
  while (true) {
    int minchild = lchild(elem);
    if (minchild >= len) {
      return;
    }
    int r = rchild(elem);
    if (r < len && h->less(h->container, r, minchild)) {
      minchild = r;
    }
    if (!h->less(h->container, minchild, elem)) {
      return;
    }
    h->swap(h->container, minchild, elem);
    elem = minchild;
  }
}

void heap_push(Heap h, void *elem) {
  h->push(h->container, elem);
  shuffleUp(h, h->len(h->container)-1);
}

void *heap_pop(Heap h) {
  return heap_delete(h, 0);
}

void *heap_delete(Heap h, int elem) {
  int n = h->len(h->container) - 1;
  if (n != elem) {
    h->swap(h->container, n, elem);
    shuffleDown(h, elem, n);
    shuffleUp(h, elem);
  }
  return h->pop(h->container);
}

void heapify(Heap h) {
  int n = h->len(h->container);
  for (int i = n / 2 - 1; i >= 0; i--) {
    shuffleDown(h, i, n);
  }
}
