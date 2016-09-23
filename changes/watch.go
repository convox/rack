package changes

import (
	"os"
	"path/filepath"
	"time"
)

func watchForChanges(dir string, ignore []string, ch chan Change) error {
	cur, err := snapshot(dir)
	if err != nil {
		return err
	}

	for {
		snap, err := snapshot(dir)
		if err != nil {
			return err
		}

		for ck, ct := range cur {
			st, ok := snap[ck]

			switch {
			case !ok:
				send(ch, "remove", ck, dir, ignore)
			case ct.Before(st):
				send(ch, "add", ck, dir, ignore)
			}
		}

		for sk := range snap {
			if _, ok := cur[sk]; !ok {
				send(ch, "add", sk, dir, ignore)
			}
		}

		cur = snap

		time.Sleep(700 * time.Millisecond)
	}

	return nil
}

func send(ch chan Change, op, file, base string, ignore []string) {
	rel, err := filepath.Rel(base, file)
	if err != nil {
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
