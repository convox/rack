#include <stdlib.h> 
#include <string.h>
#include <pthread.h>

#include "dynamic_array.h"

struct dynamic_array {
  void **elements;
  int len;
  int cap;

  void **elements_start; // points to original starting position after a slice
  int *slicecount;
  
  pthread_mutexattr_t mutexattr;
  pthread_mutex_t mutex; // uesd to update slicecount
};

dynarr *create_dynarr(void) {
  dynarr *r = malloc(sizeof(dynarr));
  r->len = 0;
  r->cap = 0;

  r->elements = malloc(0);

  r->slicecount = malloc(sizeof(int));
  *r->slicecount = 0;
  
  pthread_mutexattr_init(&r->mutexattr);
  pthread_mutex_init(&r->mutex, &r->mutexattr);
  return r;
}

dynarr *make_dynarr(int len, int cap) {
  dynarr *r = malloc(sizeof(dynarr));
  if (len < 0) {
    exit(-1);
  }
  if (cap < len) {
    cap = len;
  }
  r->len = len;
  r->cap = cap;
  r->elements = malloc(cap * sizeof(void*));
  r->elements_start = r->elements;

  r->slicecount = malloc(sizeof(int));
  *r->slicecount = 0;
  
  pthread_mutexattr_init(&r->mutexattr);
  pthread_mutex_init(&r->mutex, &r->mutexattr);
  return r;
}

void destroy_dynarr(dynarr *array) {
  pthread_mutex_lock(&array->mutex);
  if (*array->slicecount == 0) {
    free(array->slicecount);
    free(array->elements_start);
    pthread_mutex_unlock(&array->mutex);
    pthread_mutexattr_destroy(&array->mutexattr);
    pthread_mutex_destroy(&array->mutex);
    free(array);
  } else {
    (*array->slicecount)--;
    pthread_mutex_unlock(&array->mutex);
    free(array);
  }
}

void *dynarr_append(dynarr *array, void *element) {
  // Appending should only free the previous elements pointer
  // if no other slices currently contain those elements.
  // Hence, appending will never free anything that has slices of it.
  // However, if the array does need to allocate a larger capacity,
  // it will no longer belong to any previous slices.
  if (array->cap == 0) {
    void **new_array = malloc(sizeof(void*));
    if (*array->slicecount > 0) {
      pthread_mutex_lock(&array->mutex);
      (*array->slicecount)--;
      pthread_mutex_lock(&array->mutex);
      array->slicecount = malloc(sizeof(int));
      *array->slicecount = 0;
    } else {
      free(array->elements);
    }
    array->elements = new_array;
    array->elements_start = array->elements;
    array->cap = 1;
  }

  if (array->len == array->cap) {
    void **new_array;
    if (array->cap > 1000)  {
      new_array = malloc(sizeof(void*) * 1.2 * array->cap);
      array->cap = (int)(array->cap * 1.2);

    } else {
      new_array = malloc(sizeof(void*) * 2 * array->cap);
      array->cap *= 2;
    }
    memcpy(new_array, array->elements, sizeof(void*) * array->len);
    if (*array->slicecount > 0) {
      pthread_mutex_lock(&array->mutex);
      (*array->slicecount)--;
      pthread_mutex_unlock(&array->mutex);
      array->slicecount = malloc(sizeof(int));
      *array->slicecount = 0;
    } else {
      free(array->elements);
    }
    array->elements = new_array;
    array->elements_start = array->elements;
  }

  if (array->elements != NULL) {
    array->elements[array->len] = element;
  }
  array->len++;
  return array->elements;
}

dynarr *dynarr_slice(dynarr *array, int from, int to) {
  if (to > array->len) {
    exit(1);
  }
  dynarr *new = malloc(sizeof(dynarr));
  new->elements = &array->elements[from];
  // Elements start is needed if the original array is destroyed
  // before this new one. Otherwise, will not know where the original
  // elements began.
  new->elements_start = array->elements_start;
  new->len = to-from;
  new->cap = array->cap-from;
  new->slicecount = array->slicecount;
  new->mutexattr = array->mutexattr;
  new->mutex = array->mutex;
  pthread_mutex_lock(&array->mutex);
  (*array->slicecount)++;
  pthread_mutex_unlock(&array->mutex);
  return new;
}

void *dynarr_at(dynarr *array, int position) {
  return array->elements[position];
}

void dynarr_set(dynarr *array, int position, void *element) {
  array->elements[position] = element;
}

int dynarr_len(dynarr *array) {
  return array->len;
}
