#include "max_subarray.h"

max_info max_subarray_crossing(int *arr, int l, int r) {
  int m = (r + l) / 2;
  int li = m-1, lsum = arr[m-1], nsum = arr[m-1];
  for (int n = li-1; n >= l; n--) {
    nsum += arr[n];
    if (nsum > lsum) {
      lsum = nsum;
      li = n;
    }
  }
  int ri = m, rsum = arr[m];
  nsum = arr[m];
  for (int n = ri+1; n < r; n++) {
    nsum += arr[n];
    if (nsum > rsum) {
      rsum = nsum;
      ri = n;
    }
  }
  max_info best = {li, ++ri, lsum + rsum};
  return best;
}

max_info max_subarray_recursive(int *arr, int l, int r) {
  if (r - l <= 1) {
    max_info max = {l, r, 0};
    if (r == l) {
      return max;
    }
    max.sum = arr[l];
    return max;
  } else {
    max_info max_left = max_subarray_recursive(arr, l, (r+l)/2);
    max_info max_right = max_subarray_recursive(arr, (r+l)/2, r);
    max_info max_crossing = max_subarray_crossing(arr, l, r);
    if (max_left.sum > max_right.sum && max_left.sum > max_crossing.sum) {
      return max_left;
    } else if (max_right.sum > max_left.sum && max_right.sum > max_crossing.sum) {
      return max_right;
    } else {
      return max_crossing;
    }
  }
}

// maximum subarray from left to (not through) right
max_info max_subarray(int *arr, int l, int r) {
  max_info max_now = {l, l, 0};
  if (r - l <= 1) {
    max_now.r = r;
    if (r == l) {
      return max_now;
    }
    max_now.sum = arr[l];
    return max_now;
  }
  max_now.sum = arr[l];
  max_info max_so_far = {l, l, arr[l]}; 
  for (l += 1; l < r; l++) {
    if (max_now.sum + arr[l] > arr[l]) { // yet net higher than start
      max_now.r = l;
      max_now.sum += arr[l];
    } else { // new lowest low
      max_now.l = l;
      max_now.r = l;
      max_now.sum = arr[l];
    }
    if (max_now.sum > max_so_far.sum) {
      max_so_far = max_now;
    }
  }
  max_so_far.r++; // increment to one past the actual right index
  return max_so_far;
}

