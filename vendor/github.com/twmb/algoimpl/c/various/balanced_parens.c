#include <stdbool.h>

#include "balanced_parens.h"

char do_is_balanced(const char **string) {
  for (; **string != '\0'; ++*string) {
    switch (**string) {
      case '(': if (++*string, do_is_balanced(string) != ')') return '('; break;
      case '{': if (++*string, do_is_balanced(string) != '}') return '{'; break;
      case '[': if (++*string, do_is_balanced(string) != ']') return '['; break;
      case ')': 
      case ']':
      case '}':
        return **string;
    }
  }
  return **string;
}

bool is_balanced(const char *string) {
  return do_is_balanced(&string) == '\0';
}
