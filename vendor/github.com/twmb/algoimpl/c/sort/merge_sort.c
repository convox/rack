#include "merge_sort.h"

typedef struct {
  int start;
  int end;
} merge_segment;

// combines the left and right segments from scrambled into sorted
// sort from start bounds to end bounds into sorted, then copy
// correct positions back into scrambled (this could be improved, obviously)
void merge_combine(int *scrambled, int *sorted,
    merge_segment l, merge_segment r) {
  int li=l.start, ri=r.start, ci=l.start;  // left start always < right start
  while (li < l.end && ri < r.end) {
    if (scrambled[li] < scrambled[ri]) {
      sorted[ci++] = scrambled[li++];
    } else {
      sorted[ci++] = scrambled[ri++];
    }
  }
  while (li < l.end) {
    sorted[ci++]=scrambled[li++];
  }
  while (ri < r.end) {
    sorted[ci++]=scrambled[ri++];
  }
  for (ci = l.start; ci < r.end; ci++) {
    scrambled[ci] = sorted[ci];
  }
}

// scrambled needs to be sorted, sorted is the target int array
merge_segment merge_sort_impl(int *scrambled, int *sorted, int from, int to) {
  if (to-1 > from) {
    merge_segment left = merge_sort_impl(scrambled, sorted, from, (to+from)/2);
    merge_segment right = merge_sort_impl(scrambled, sorted, (to+from)/2, to);
    merge_combine(scrambled, sorted, left, right);
  }
  merge_segment n = {from,to};
  return n;
}

void merge_sort(int *scrambled, int from, int to){
  int sorter[to-from];
  merge_sort_impl(scrambled, sorter, from, to);
}
