package changes

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/inotify"
)

var (
	dirCreateFlags       = inotify.IN_CREATE | inotify.IN_ISDIR
	dirDeleteFlags       = inotify.IN_DELETE | inotify.IN_ISDIR
	watcher              *inotify.Watcher
	lock                 sync.Mutex
	fallbackSyncTickTime = fallbackSyncTickTimeInMillis()
	touchTimes           = map[string](time.Time){}
)

func init() {
	watcher, _ = inotify.NewWatcher()
}

// TODO: pass ignore to this func
func startScanner(dir string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info != nil && info.IsDir() {
			lock.Lock()
			watcher.AddWatch(path, inotify.IN_CREATE|inotify.IN_DELETE|inotify.IN_MODIFY|inotify.IN_ATTRIB)
			lock.Unlock()
		}
		return nil
	})
}

// Wait for a file system event, then return. The caller func (see changes.go
// watchForChanges ) will then Walk the dir and sync any file changes that it
// detects.
func waitForNextScan(dir string) {
	tick := time.Tick(fallbackSyncTickTime)

	for {
		select {
		case ev := <-watcher.Event:

			if strings.HasPrefix(ev.Name, dir) {

				touchTimes[dir] = time.Now()

				if ev.Mask|dirCreateFlags == dirCreateFlags {
					startScanner(ev.Name)
				}
				if ev.Mask|dirDeleteFlags == dirDeleteFlags {
					watcher.RemoveWatch(ev.Name)
				}

				// Force a brief wait, since many editors may send events in a burst of
				// activity that is over w/in a few millis.
				time.Sleep(100 * time.Millisecond)

				return
			}
		case <-tick:
			// Force a resync on changed files on each tick if dir is hot
			if isHot(dir) {
				return
			}
		}
	}
}

// Return whether dir has received any inotify events in last 10 minutes.
func isHot(dir string) bool {
	ttime := touchTimes[dir]
	elapsedMillis := time.Since(ttime) / 1000000
	return (elapsedMillis < 600000)
}

func fallbackSyncTickTimeInMillis() time.Duration {
	ttime := 2000
	tickString := os.Getenv("FALLBACK_SYNC_TICK")
	if tickString != "" {
		t, _ := strconv.ParseInt(tickString, 0, 32)
		ttime = int(t)
	}
	return (time.Duration(ttime) * time.Millisecond)
}

func isDebugging() bool {
	return os.Getenv("CONVOX_DEBUG") != ""
}
