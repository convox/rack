#include <stdio.h>
#include <string.h>
#include <unistd.h>

#include "permute_letters.h"

#define BUFLEN 40

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

  permute_letters("hat", 3);
  // Expect
  //`hat
  //`hta
  //`aht
  //`ath
  //`tha
  //`tah
  //`
  fflush(stdout);

  read(pipefds[0], buffer, BUFLEN);
  if (strcmp(buffer, "hat\nhta\naht\nath\ntha\ntah\n")) {
    failed = -1;
    printf("error, buffer %s not equal to 'hat\nhta\naht\nath\ntha\ntah\n'", buffer);
  }

  dup2(stdoutfd, STDOUT_FILENO);
  return failed;
}

