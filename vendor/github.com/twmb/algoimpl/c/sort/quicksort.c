#include <stdbool.h>
#include <stdlib.h>

#include "quicksort.h"

// XOR swap algorithm
static void swap(int *left, int *right) {
  if (left != right) {
    *left ^= *right;
    *right ^= *left;
    *left ^= *right;
  }
}

// http://biobio.loc.edu/chu/web/Courses/COSI216/median3.htm
// see link for explanation
// TODO: change to use bubblesort, median of median of threes like
// http://golang.org/src/pkg/sort/sort.go?s=4433:4458#L182
static int median_of_three_ints(int *array, int len) {
  len--;
  int indices[3] = {0, len/2, len};
  int small = array[0], large = array[len/2], final = array[len];
  if (small > large) {
    swap(&small, &large);
    swap(&indices[0], &indices[1]);
  }
  if (final < small) {
    return indices[0];
  }
  if (final > large) {
    return indices[1];
  }
  return indices[2];
}

// Performs the quicksort algorithm assuming the pivot
// is at the end. It returns the index of the pivot after
// sorting.
static int do_quicksort(int *array, size_t len) {
  int lessBarrier = 1;
  for (int i = 1; i < len; i++) {
    if (array[i] < array[0]) {
      swap(&array[i], &array[lessBarrier]);
      lessBarrier++;
    }
  }
  swap(&array[0], &array[lessBarrier-1]);
  return lessBarrier-1;
}

void quicksort_ints(int *array, size_t nmemb) {
  // Tail call quicksort
  while (nmemb > 1) {
    int pivotIndex = median_of_three_ints(array, nmemb);
    swap(&array[pivotIndex], &array[0]);
    pivotIndex = do_quicksort(array, nmemb);
    quicksort_ints(&array[pivotIndex+1], nmemb - 1 - pivotIndex);
    nmemb = pivotIndex;
  }
}

// http://biobio.loc.edu/chu/web/Courses/COSI216/median3.htm
// see link for explanation
//static int median_of_three(void *array, int len, (*less)(void *left, void *right)) {
//  len--;
//  bool swapped = false;
//  int small = indices[0], large = indices[1], final = indices[2];
//  if (less(large, small)  large) {
//    swap(&small, &large);
//    swap(&indices[0], &indices[1]);
//  }
//  if (final < small) {
//    return indices[0];
//  }
//  if (final > large) {
//    return indices[1];
//  }
//  return indices[2];
//}
//
//void quicksort(void *array,
//    bool (*less)(void *left, void *right),
//    void (*swap)(void *left, void *right),
//    int (*len)(void *array)) {
//  while (len(array) > 0) {
//    int pivotIndedx = median_of_three(array, len(array));
//
//  }
//}
//
