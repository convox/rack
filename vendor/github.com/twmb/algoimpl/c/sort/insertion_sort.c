#include "insertion_sort.h"

void insertion_sort(int *arr, int start, int length) {
  int current, i;
  for (int j = start+1; j < length; j++) {
    current = arr[j]; // save copy of number we are inserting
    for (i = j-1; i >= start && arr[i] > current; i--) {
      if (arr[i] > current) { 
        arr[i+1]=arr[i]; // slide larger left int right one
      }
    }
    arr[i] = current; // insert into position
  }
}

