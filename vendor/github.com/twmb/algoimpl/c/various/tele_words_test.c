#include <stdio.h>
#include <string.h>
#include <unistd.h>

#include "tele_words.h"

#define BUFLEN 20

int main() {
  int failed = 0;

  char buffer[] = {[BUFLEN] = '\0'};
  int pipefds[2];
  int stdoutfd = dup(STDOUT_FILENO);
  if (pipe(pipefds) != 0) {
    return -1;
  }
  dup2(pipefds[1], STDOUT_FILENO);
  close(pipefds[1]);

  phone_number_letters("012", 3);
  // Expect
  //`YZA
  //`YZB
  //`YZC
  //`
  fflush(stdout);

  read(pipefds[0], buffer, BUFLEN);
  if (strcmp(buffer, "YZA\nYZB\nYZC\n")) {
    failed = -1;
    printf("error, buffer %s not equal to 'YZA\nYZB\nYZC\n'", buffer);
  }

  dup2(stdoutfd, STDOUT_FILENO);
  return failed;
}
