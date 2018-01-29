#include <stdio.h>
#include <string.h>
#include <unistd.h>

#include "combination_letters.h"

#define BUFLEN 100 

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

  combine_letters("wxyz", 4);
  fflush(stdout);
  char *expect = "w\nwx\nwxy\nwxyz\nwxz\nwy\nwyz\nwz\nx\nxy\nxyz\nxz\ny\nyz\nz\n";

  read(pipefds[0], buffer, BUFLEN);
  if (strcmp(buffer, expect)) {
    failed = -1;
    printf("error, buffer %s not equal to '%s'\n", buffer, expect);
  }

  dup2(stdoutfd, STDOUT_FILENO);
  return failed;
}

