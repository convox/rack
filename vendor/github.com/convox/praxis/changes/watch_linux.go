package changes

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/convox/inotify"
)

var (
	dirCreateFlags = inotify.IN_CREATE | inotify.IN_ISDIR
	dirDeleteFlags = inotify.IN_DELETE | inotify.IN_ISDIR
	watcher        *inotify.Watcher
	lock           sync.Mutex
)

func init() {
	watcher, _ = inotify.NewWatcher()
}

func startScanner(dir string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info != nil && info.IsDir() {
			lock.Lock()
			// watcher.AddWatch(path, inotify.IN_CREATE|inotify.IN_DELETE|inotify.IN_MODIFY|inotify.IN_ATTRIB)
			watcher.Watch(path)
			lock.Unlock()
		}
		return nil
	})
}

func waitForNextScan(dir string) {
	tick := time.Tick(900 * time.Millisecond)
	fired := false

	for {
		select {
		case ev := <-watcher.Event:
			if strings.HasPrefix(ev.Name, dir) {
				if ev.Mask|dirCreateFlags == dirCreateFlags {
					startScanner(ev.Name)
				}
				if ev.Mask|dirDeleteFlags == dirDeleteFlags {
					watcher.RemoveWatch(ev.Name)
				}
				fired = true
			}
		case <-tick:
			if fired {
				return
			}
		}
	}
}
