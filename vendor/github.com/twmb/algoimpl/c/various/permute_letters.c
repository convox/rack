#include <stdio.h>
#include <stdbool.h>
#include <string.h>

#include "permute_letters.h"

void do_permute_letters(const char *letters, int len, bool *used, char *buffer, int position) {
  if (position == len) {
    printf("%s\n", buffer);
    return;
  }
  for (int i = 0; i < len; i++) {
    if (!used[i]) {
      buffer[position] = letters[i];
      used[i] = true;
      do_permute_letters(letters, len, used, buffer, position+1);
      used[i] = false;
    }
  }
}

void permute_letters(const char *letters, int len) {
  char buffer[len+1];
  buffer[len] = '\0';
  strcpy(buffer, letters);
  bool used[len];
  for (int i = 0; i < len; i++) {
    used[i] = false;
  }
  do_permute_letters(letters, len, used, buffer, 0);
}
