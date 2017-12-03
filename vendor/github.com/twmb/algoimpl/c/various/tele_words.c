#include <stdio.h>

#include "tele_words.h" 

//struct stringNums {
//  char **strings;
//  int len;
//};
//
//struct stringNums phone_number_letters(const char *number, int len) {
//  struct stringNums r;
//  if (len == 1) {
//    if (number[0] == '0' || number[0] == '1') {
//      r.len = 1;
//      r.strings = malloc(sizeof(char *));
//      r.strings[0] = malloc(sizeof(char));
//      r.strings[0][0] = get_char_key(number[0] - '0', 0);
//    } else {
//      r.len = 3;
//      r.strings = malloc(3 * sizeof(char *));
//      for (int i = 0; i < 3; i++) {
//        r.strings[i] = malloc(sizeof(char));
//        r.strings[i][0] = get_char_key(number[0] - '0', i);
//      }
//    }
//  } else {
//    struct stringNums substrings = first_phone_number_letters(number + 1, len - 1);
//    if (number[0] == '0' || number[0] == '1') {
//      r.len = substrings.len;
//      r.strings = malloc(substrings.len * sizeof(char *));
//      for (int j = 0; j < substrings.len; j++) {
//        r.strings[j] = malloc(len * sizeof(char));
//        strcpy(r.strings[j] + 1, substrings.strings[j]);
//        r.strings[j][0] = get_char_key(number[0] - '0', 0);
//      }
//    } else {
//      r.len = 3 * substrings.len;
//      r.strings = malloc(3 * substrings.len * sizeof(char *));
//      for (int i = 0; i < 3; i++) {
//        for (int j = 0; j < substrings.len; j++) {
//          r.strings[i*substrings.len + j] = malloc(len * sizeof(char));
//          strcpy(r.strings[i*substrings.len + j] + 1, substrings.strings[j]);
//          r.strings[i*substrings.len + j][0] = get_char_key(number[0] - '0', i);
//        }
//      }
//    }
//    for (int i = 0; i < substrings.len; i++) {
//      free(substrings.strings[i]);
//    }
//    free(substrings.strings);
//  }
//  return r;
//}

char get_char_key(int number, int place) {
  switch (number) {
    case 0:
      return 'Y'; break;
    case 1:
      return 'Z'; break;
    default:
      return 'A' + (number - 2) * 3 + place;
  }
}

void do_phone_number_letters(char *number, int len, char *buffer) {
  if (*buffer == '\0') {
    printf("%s\n", buffer - len);
    return;
  }
  for (int i = 0; i < 3; i++) {
    buffer[0] = get_char_key(number[0] - '0', i);
    do_phone_number_letters(number + 1, len, buffer + 1);
    if (number[0] == '0' || number[0] == '1') {
      return;
    }
  }
}

void phone_number_letters(char *number, int len) {
  char buffer[len + 1];
  for (int i = 0; i < len; i++) {
    // initialize to not be null
    buffer[i] = 'a';
  }
  buffer[len] = '\0';
  do_phone_number_letters(number, len, buffer);
}
