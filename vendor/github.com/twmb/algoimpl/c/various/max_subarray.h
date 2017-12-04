#ifndef TWMB_MAX_SUBARRAY
#define TWMB_MAX_SUBARRAY

typedef struct {
  int l;
  int r;
  int sum;
} max_info;

max_info max_subarray_recursive(int *arr, int l, int r);
max_info max_subarray(int *arr, int l, int r);

#endif
