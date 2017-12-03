#include <stdio.h>

#include "balanced_parens.h"

int main() {
  int failed = 0;

  if (!is_balanced("asdf")) {
    failed = -1;
    printf("failed on input \"asdf\"\n");
  }
  if (!is_balanced("({})")) {
    failed = -1;
    printf("failed on input \"({})\"\n");
  }
  char *str = "asdf(asdf{asdf{asdf}asdf[asdf(asdf)]})";
  if (!is_balanced(str)) {
    failed = -1;
    printf("failed on input \"%s\"\n", str);
  }
  if (is_balanced("(")) {
    failed = -1;
    printf("failed on input \"(\"\n");
  }
  if (is_balanced("a(sdf])")) {
    failed = -1;
    printf("failed on input \"a(sdf])\"\n");
  }
  return failed;
}
