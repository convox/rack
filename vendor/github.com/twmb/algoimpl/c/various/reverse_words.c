#include <string.h>

static void reverse_string(char *string, int start, int end) {
  char tmp;
  while (end > start) {
    tmp = string[start];
    string[start] = string[end];
    string[end] = tmp;
    end--; start++;
  }
}

void reverse_words(char line[]) {
  int len = strlen(line);
  if (len < 1) {
    return;
  }
  if (line[len-1] == '\n') {
    reverse_string(line,0,len-2);
  } else {
    reverse_string(line,0,len-1);
  }
  int wordstart = 0, wordend = 0;
  for (; line[wordend] != '\0' && line[wordend] != '\n'; wordend++) {
    if(line[wordend] == ' ') {
      reverse_string(line, wordstart, wordend-1);
      wordstart = wordend + 1;
    }
  }
  reverse_string(line, wordstart, wordend-1);
}
