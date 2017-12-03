#include <stdio.h>

#include "merge_sort.h"

int main(int argv, char **argc) {
  int failed = 0;
  int us1[1] = {-1};
  int us2[5] = {4,3,2,1,2};
  merge_sort(us1, 0, 1);
  merge_sort(us2, 0, 5);
  int prev = us2[0];
  for (int i = 0; i < 5; i++) {
    if (us2[i] < prev) {
      printf("us2 error: index %d, val %d > index %d, val %d\n", i, us2[i], i-1, prev);
      failed = -1;
    }
  }
  return failed;
}
