#ifndef TWMB_QUICKSORT
#define TWMB_QUICKSORT

#include <stdbool.h>

// Runs quicksort on an array of ints of length len.
// Uses median of three for choosing the pivot.
void quicksort_ints(int *array, size_t nmemb);

// TODO: implement
// Runs quicksort on a generic array of elements.
// It requires three function pointers to operate
// on this array.
//void quicksort(void *array, 
//  // takes a left and right element and returns
//  // if the left is smaller than the right
//  bool (*less)(void *left, void *right),
//  // takes a left and right element and swaps them
//  void (*swap)(void *left, void *right),
//  // returns the length of the array
//  int (*len)(void));

#endif
