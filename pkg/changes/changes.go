package changes

import (
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
	cur, err := snapshot(dir)
	if err != nil {
		return err
	}

	startScanner(dir)

	for {
		snap, err := snapshot(dir)
		if err != nil {
			return err
		}

		notify(ch, cur, snap, dir, ignore)

		cur = snap

		waitForNextScan(dir)
	}

	return nil
}

func notify(ch chan Change, from, to map[string]time.Time, base string, ignore []string) {
	for fk, ft := range from {
		tt, ok := to[fk]

		switch {
		case !ok:
			send(ch, "remove", fk, base, ignore)
		case ft.Before(tt):
			send(ch, "add", fk, base, ignore)
		}
	}

	for tk := range to {
		if _, ok := from[tk]; !ok {
			send(ch, "add", tk, base, ignore)
		}
	}
}

func send(ch chan Change, op, file, base string, ignore []string) {
	rel, err := filepath.Rel(base, file)
	if err != nil {
		return
	}

	if match, _ := fileutils.Matches(rel, ignore); match {
		return
	}

	change := Change{
		Operation: op,
		Base:      base,
		Path:      rel,
	}

	ch <- change
}

func snapshot(dir string) (map[string]time.Time, error) {
	snap := map[string]time.Time{}

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
