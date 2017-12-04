#include <stdio.h>

#include "combination_letters.h"

void do_combine_letters(const char *letters, char *printer, int printer_len, int letters_len, int start_letter) {
  for (int i = start_letter; i < letters_len; i++) {
    printer[printer_len] = letters[i];
    printf("%s\n", printer);
    if (i < letters_len-1) {
      do_combine_letters(letters, printer, printer_len+1, letters_len, i+1);
    }
    printer[printer_len] = '\0';
  }
}

void combine_letters(const char *letters, int len) {
  char printer[len];
  for (int i = 0; i < len; i++) {
    printer[i] = '\0';
  }
  do_combine_letters(letters, printer, 0, len, 0);
}


