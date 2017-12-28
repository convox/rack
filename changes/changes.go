package changes

import (

	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/builder/dockerignore"
	"github.com/docker/docker/pkg/fileutils"
)

type Change struct {
	Operation string
	Base      string
	Path      string
}

type dirSnapshot map[string]time.Time

func Partition(changes []Change) (adds []Change, removes []Change) {
	for _, c := range changes {
		switch c.Operation {
		case "add":
			adds = append(adds, c)
		case "remove":
			removes = append(removes, c)
		}
	}

	return
}

func Watch(dir string, ch chan Change) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	ignore, err := readDockerIgnoreRecursive(abs)
	if err != nil {
		return err
	}

	sym, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return err
	}

	return watchForChanges(sym, ignore, ch)
}

func readDockerIgnoreRecursive(root string) ([]string, error) {
	ignore := []string{}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info != nil && info.Name() == ".dockerignore" {
			lines, err := readDockerIgnore(path)
			if err != nil {
				return err
			}

			// get the relative base between the root of the docker context and this dockerignore
			rel, err := filepath.Rel(root, filepath.Dir(path))
			if err != nil {
				return err
			}

			for _, line := range lines {
				// append the dockerignore lines including the relative base
				ignore = append(ignore, filepath.Join(rel, line))
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return ignore, nil
}

func readDockerIgnore(file string) ([]string, error) {
	fd, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	ignore, err := dockerignore.ReadAll(fd)
	if err != nil {
		return nil, err
	}

	return ignore, nil
}

func watchForChanges(dir string, ignore []string, ch chan Change) error {
	prev, err := snapshot(dir)
	if err != nil {
		return err
	}

	startScanner(dir)

	for {

		prev, err = syncUntilStable(dir, ignore, ch, prev)
		if err != nil {
			return err
		}

		waitForNextScan(dir)

	}

	return nil
}

// Do a code-sync (notify), then compare the dir contents to the contents from
// before the sync. Repeat until dir contents dont change for at least 200ms
// Give up on stabilizing dir after 100 syncs. Also see func comment for
// watchForChanges. Return the final dir snapshot.
func syncUntilStable(dir string, ignore []string, ch chan Change, prev dirSnapshot) (dirSnapshot, error) {
	for i := 0; i < 10; i++ {

		snap, err := snapshot(dir)
		if err != nil {
			return prev, err
		}

		changed := notify(ch, prev, snap, dir, ignore)

		if isDebugging() && changed && i > 0 {
			fmt.Printf("syncUntilStable: multipass (%s) change: %s ... ", i, changed)
		}

		prev = snap

		if changed {
			// wait a bit, then resnap to look for more changes
			time.Sleep(200 * time.Millisecond)
		} else {
			return prev, nil
		}

	}

	fmt.Printf("syncUntilStable: dir never stabilized %s", dir)
	return prev, nil
}

func notify(ch chan Change, from, to dirSnapshot, base string, ignore []string) bool {

	changed := false

	for fk, ft := range from {
		tt, ok := to[fk]

		switch {
		case !ok:
			changed = send(ch, "remove", fk, base, ignore) || changed
		case ft.Before(tt):
			changed = send(ch, "add", fk, base, ignore) || changed
		}
	}

	for tk := range to {
		if _, ok := from[tk]; !ok {
			changed = send(ch, "add", tk, base, ignore) || changed
		}
	}

	return changed
}

func send(ch chan Change, op, file, base string, ignore []string) bool {
	rel, err := filepath.Rel(base, file)
	if err != nil {
		return false
	}

	if match, _ := fileutils.Matches(rel, ignore); match {
		return false
	}

	change := Change{
		Operation: op,
		Base:      base,
		Path:      rel,
	}

	ch <- change
	return true
}

func snapshot(dir string) (dirSnapshot, error) {
	snap := dirSnapshot{}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		snap[path] = info.ModTime()
		return nil
	})
	if err != nil {
		return nil, err
	}

	return snap, nil
}

func isDebugging() bool {
	return os.Getenv("CONVOX_DEBUG") == "true"
}
