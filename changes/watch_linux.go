package changes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/convox/inotify"
)

var (
	dirCreateFlags	= inotify.IN_CREATE | inotify.IN_ISDIR
	dirDeleteFlags	= inotify.IN_DELETE | inotify.IN_ISDIR
	watcher			*inotify.Watcher
	lock			sync.Mutex
)

func init() {
	watcher, _ = inotify.NewWatcher()
}

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

	var fallbackSyncTick <-chan time.Time

	if isFallbackSyncOn() {
		fallbackSyncTick = time.Tick(fallbackSyncTickTime)
	}

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

				if isDebugging() {
					fmt.Printf("waitForNextScan Event: (%s) ", dir)
				}

				return
			}
		case <-fallbackSyncTick:
			if isHot(dir) {
				return
			}
		}
	}
}

