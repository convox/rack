#ifndef TWMB_HEAP_C
#define TWMB_HEAP_C

#include <stdbool.h>

// A Heap defines an interface for using heap functions.
// To use a Heap, first call create_heap followed by the
// set functions. After that, either call heapify on your
// heap that has existing elements in the container or use
// the heap normally.
typedef struct heap *Heap;

// Pushes an element onto the heap.
void heap_push(Heap h, void *elem);
// Pops and returns an element from the heap.
void *heap_pop(Heap h);
// Removes and returns the element at the specified index.
void *heap_delete(Heap h, int elem);

// Creates and returns an empty heap.
// Initialize it with the set_heap functions before using the heap.
Heap create_heap(void);
// Creates a heap out of an unordered container.
void heapify(Heap h);
// Destroys the heap.
void destroy_heap(Heap h);

// Sets the abstract collection that the heap will use.
void set_heap_container(Heap h, void *container);
// Less compares two elements in the collection.
void set_heap_lessfunc(Heap h, bool (*less)(void *container, int left, int right));
// Len returns the length of the collection. 
void set_heap_lenfunc(Heap h, int (*len)(void *container));
// Swap swaps two elements in the collection.
void set_heap_swapfunc(Heap h, void (*swap)(void *container, int left, int right));
// Push adds an element to the end of the collection.
void set_heap_pushfunc(Heap h, void (*push)(void *container, void *elem));
// Pop removes and returns the element from the end of the collection.
void set_heap_popfunc(Heap h, void *(*pop)(void *container));

#endif
