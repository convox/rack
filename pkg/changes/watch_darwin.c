// +build darwin,cgo

#include <CoreServices/CoreServices.h>
#include "_cgo_export.h"

void
fswatch_callback(ConstFSEventStreamRef streamRef,
                 void *clientCallBackInfo,
                 size_t numEvents,
                 void *eventPaths,
                 const FSEventStreamEventFlags eventFlags[],
                 const FSEventStreamEventId eventIds[])
{
  callback(
      (FSEventStreamRef)streamRef,
      clientCallBackInfo,
      numEvents,
      eventPaths,
      (FSEventStreamEventFlags*)eventFlags,
      (FSEventStreamEventId*)eventIds);
}

FSEventStreamRef fswatch_create(FSEventStreamContext *ctx, CFMutableArrayRef pathsToWatch, FSEventStreamEventId since, CFTimeInterval latency, FSEventStreamCreateFlags flags) {
  return FSEventStreamCreate(
      NULL,
      fswatch_callback,
      ctx,
      pathsToWatch,
      since,
      latency,
      flags);
}
