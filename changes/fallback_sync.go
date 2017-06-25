package changes

/**
  * Fallback sync is off unless the env var FALLBACK_SYNC_TICK is defined.
  * 
  * Fallback sync will cause a directory to be code-synced every
  * FALLBACK_SYNC_TICK milliseconds if that directory is 'hot'. A directory is
  * hot if some editing event (create, modify) has been detected in the last 10
  * minutes somewhere under that directory.
  * 
  * Fallback sync does not appear to be necessary except in the cases where FS
  * events can get lost. Eg. when convox is running on linux on a file
  * system mounted over NFS or ssh fuse.
*/

import (
	"os"
	"strconv"
	"time"
)

var (
	fallbackSyncTickTime = fallbackSyncTickTimeInMillis()
	touchTimes           = map[string](time.Time){}
)

// Return whether dir has received any inotify events in last 10 minutes.
func isHot(dir string) bool {
	ttime := touchTimes[dir]
	elapsedMillis := time.Since(ttime) / 1000000
	return (elapsedMillis < 60000)
}

func fallbackSyncTickTimeInMillis() time.Duration {
	ttime := 5000 // 5s
	tickString := os.Getenv("FALLBACK_SYNC_TICK")
	if tickString != "" {
		t, _ := strconv.ParseInt(tickString, 0, 32)
		ttime = int(t)
	}
	return (time.Duration(ttime) * time.Millisecond)
}

func isFallbackSyncOn() bool {
	return os.Getenv("FALLBACK_SYNC_TICK") != ""
}

