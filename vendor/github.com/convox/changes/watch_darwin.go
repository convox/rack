// +build darwin,cgo

// heavily inspired by https://github.com/fsnotify/fsevents

package changes

/*
#cgo LDFLAGS: -framework CoreServices
#include <CoreServices/CoreServices.h>
FSEventStreamRef fswatch_new(
	FSEventStreamContext*,
	CFMutableArrayRef,
	FSEventStreamEventId,
	CFTimeInterval,
	FSEventStreamCreateFlags);
static CFMutableArrayRef fswatch_make_mutable_array() {
  return CFArrayCreateMutable(NULL, 0, &kCFTypeArrayCallBacks);
}
*/
import "C"

import (
	"math/rand"
	"sync"
	"time"
	"unsafe"
)

var (
	cflags   = C.FSEventStreamCreateFlags(0)
	chans    = make(map[string](chan string))
	interval = 700 * time.Millisecond
	now      = C.FSEventStreamEventId((1 << 64) - 1)

	lock sync.Mutex
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func startScanner(dir string) {
	lock.Lock()
	chans[dir] = make(chan string)
	lock.Unlock()

	cpaths := C.fswatch_make_mutable_array()
	defer C.free(unsafe.Pointer(cpaths))

	path := C.CString(dir)
	str := C.CFStringCreateWithCString(nil, path, C.kCFStringEncodingUTF8)
	defer C.free(unsafe.Pointer(path))
	defer C.free(unsafe.Pointer(str))

	C.CFArrayAppendValue(cpaths, unsafe.Pointer(str))

	ctx := C.FSEventStreamContext{info: unsafe.Pointer(C.CString(dir))}

	stream := C.fswatch_new(&ctx, cpaths, now, C.CFTimeInterval(interval/time.Second), cflags)

	go func() {
		C.FSEventStreamScheduleWithRunLoop(stream, C.CFRunLoopGetCurrent(), C.kCFRunLoopCommonModes)
		C.FSEventStreamStart(stream)
		C.CFRunLoopRun()
	}()
}

func waitForNextScan(dir string) {
	tick := time.Tick(900 * time.Millisecond)
	fired := false

	for {
		lock.Lock()
		ch, ok := chans[dir]
		lock.Unlock()

		if !ok {
			return
		}

		select {
		case <-ch:
			tick = time.Tick(500 * time.Millisecond)
			fired = true
		case <-tick:
			if fired {
				return
			}
		}
	}
}

//export cb
func cb(stream C.FSEventStreamRef, info unsafe.Pointer, count C.size_t, paths **C.char, flags *C.FSEventStreamEventFlags, ids *C.FSEventStreamEventId) {
	dir := C.GoString((*C.char)(info))

	lock.Lock()
	ch, ok := chans[dir]
	lock.Unlock()

	if ok {
		ch <- ""
	}
}
