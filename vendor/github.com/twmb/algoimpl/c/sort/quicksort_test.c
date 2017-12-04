#include <stdio.h>
#include <stdbool.h>

#include "quicksort.h"

struct test {
  int *in, *want, sizes;
};

bool intsEqual(int *left, int *right, int len) {
  for (int i = 0; i < len; i++) {
    if (left[i] != right[i]) {
      return false;
    }
  }
  return true;
}

int main(void) {
  int rVal = 0;

  int t0[] = {0};
  int e0[] = {0};
  int t1[] = {-1,-1,-1};
  int e1[] = {-1,-1,-1};
  int t2[] = {3,2,1,0,-1,-2,-3};
  int e2[] = {-3,-2,-1,0,1,2,3};
  int t3[] = {-3,3,-2,2,-1,1,0};
  int e3[] = {-3,-2,-1,0,1,2,3};
  int t4[] = {5,4,1,7,6};
  int e4[] = {1,4,5,6,7};

  int testcount = 5;
  struct test tests[] = {
    {t0, e0, 1},
    {t1, e1, 3},
    {t2, e2, 7},
    {t3, e3, 7},
    {t4, e4, 5},
  };

  for (int i = 0; i < testcount; i++) {
    quicksort_ints(tests[i].in, tests[i].sizes);
    if (!intsEqual(tests[i].in, tests[i].want, tests[i].sizes)) {
      fprintf(stderr, "quicksort failed, output: ");
      for (int j = 0; j < tests[i].sizes; j++) {
        fprintf(stderr, "%d ", tests[i].in[j]);
      }
      fprintf(stderr, "\n");
      rVal = -1;
    }
  }
  return rVal;
}
