#include <stdio.h>
#include <stdint.h>

#include "dynamic_array.c"

int check_dynarr(dynarr *d, int exp_len, int exp_cap) {
  int rval;
  if (d->len != exp_len) {
    printf("failed at creating length %d, len = %d\n", exp_len, d->len);
    rval = -1;
  }
  if (d->cap != exp_cap) {
    printf("failed at creating cap %d, cap = %d\n", exp_cap, d->cap);
    rval = -1;
  }
  return rval;
}

int test_makedynarr() {
  int rval = 0;
  dynarr *d = make_dynarr(3, 4);
  rval |= check_dynarr(d, 3, 4);
  destroy_dynarr(d);

  d = make_dynarr(5, -1);
  rval |= check_dynarr(d, 5, 5);
  destroy_dynarr(d);

  d = make_dynarr(0, 0);
  rval |= check_dynarr(d, 0, 0);
  destroy_dynarr(d);

  return rval;
}

int test_dynarr_append() {
  int rval = 0;
  dynarr *d = create_dynarr();
  for (int64_t i = 0; i < 20; i++) {
    dynarr_append(d, (void *)i);
    if (i != (int64_t)dynarr_at(d, i)) {
      printf("dynarr_append: expected val %lu != actual %lu\n",
          i, (int64_t)dynarr_at(d, i));
      rval = -1;
    }
  }
  destroy_dynarr(d);
  return rval;
}

int test_dynarr_slice() {
  int rval = 0;

  dynarr *d = create_dynarr();
  for (int64_t i = 0; i < 20; i++) {
    dynarr_append(d, (void *)i);
  }

  dynarr *n = dynarr_slice(d, 10, 15);
  for (int64_t i = 0; i < dynarr_len(n); i++) {
    if (i + 10 != (int64_t)dynarr_at(n, i)) {
      printf("dynarr_slice: expected val %lu != actual %lu\n",
          i + 10, (int64_t)dynarr_at(n, i));
      rval = -1;
    }
  }
  destroy_dynarr(d);
  destroy_dynarr(n);

  return rval;
}

// dynarr_at tested with other functions

int test_dynarr_set() {
  int rval = 0;

  dynarr *d = create_dynarr();
  for (int64_t i = 0; i < 20; i++) {
    dynarr_append(d, (void *)i);
    dynarr_set(d, i, (void *)(i*i));

    if (i * i != (int64_t)dynarr_at(d, i)) {
      printf("dynarr_set: expected val %lu != actual %lu\n",
          i * i, (int64_t)dynarr_at(d, i));
      rval = -1;
    }
  }
  destroy_dynarr(d);

  return rval;
}

int test_dynarr_len() {
  int rval = 0;

  dynarr *d0 = create_dynarr();
  for (int64_t i = 0; i < 20; i++) {
    dynarr_append(d0, (void *)i);
  }
  if (20 != dynarr_len(d0)) {
    printf("dynarr_len: expected length 20 for d0, actual %d\n", dynarr_len(d0));
    rval = -1;
  }

  dynarr *d1 = dynarr_slice(d0, 10, 15);
  destroy_dynarr(d0);
  if (5 != dynarr_len(d1)) {
    printf("dynarr_len: expected length 5 for d1, actual %d\n", dynarr_len(d1));
    rval = -1;
  }
  destroy_dynarr(d1);

  dynarr *d2 = make_dynarr(5, 10);
  if (5 != dynarr_len(d2)) {
    printf("dynarr_len: expected length 5 for d2, actual %d\n", dynarr_len(d2));
    rval = -1;
  }
  destroy_dynarr(d2);

  return rval;
}

int main(void) {
  int rval = 0;
  rval |= test_makedynarr();
  rval |= test_dynarr_append();
  rval |= test_dynarr_slice();
  rval |= test_dynarr_set();
  rval |= test_dynarr_len();
  return rval;
}
