#ifndef CONFIG_H
#define CONFIG_H

#ifdef OS_LINUX
  #ifndef HIDRAW
    #define POLL_POSIX
    #define THREADS_POSIX
    #define HAVE_CLOCK_GETTIME
  #endif
#endif

#ifdef OS_DARWIN
  #define POLL_POSIX
  #define THREADS_POSIX
#endif

#if OS_WINDOWS
  #define POLL_WINDOWS
  #define THREADS_WINDOWS
#endif

#endif /* CONFIG_H */
