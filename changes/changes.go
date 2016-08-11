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
	files := map[string]map[string]time.Time{}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	ignore, err := readDockerIgnore(abs)
	if err != nil {
		return err
	}

	sym, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return err
	}

	files[sym] = map[string]time.Time{}

	err = filepath.Walk(sym, func(path string, info os.FileInfo, err error) error {
		if path == sym {
			return nil
		}

		if info != nil {
			files[sym][path] = info.ModTime()
		}

		return nil
	})

	if err != nil {
		return err
	}

	return watchForChanges(files, sym, ignore, ch)
}

func readDockerIgnore(dir string) ([]string, error) {
	fd, err := os.Open(filepath.Join(dir, ".dockerignore"))

	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	ignore, err := dockerignore.ReadAll(fd)
	if err != nil {
		return nil, err
	}

	return ignore, nil
}

func watchForChanges(files map[string]map[string]time.Time, dir string, ignore []string, ch chan Change) error {
	for {
		for file, _ := range files[dir] {
			if _, err := os.Stat(file); os.IsNotExist(err) {
				rel, err := filepath.Rel(dir, file)

				if err != nil {
					return err
				}

				delete(files[dir], file)

				ch <- Change{
					Operation: "remove",
					Base:      dir,
					Path:      rel,
				}
			}
		}

		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if path == dir {
				return nil
			}

			if info.IsDir() {
				return nil
			}

			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}

			match, err := fileutils.Matches(rel, ignore)
			if err != nil {
				return err
			}

			if match {
				return nil
			}

			e, ok := files[dir][path]

			if !ok || e.Before(info.ModTime()) {
				rel, err := filepath.Rel(dir, path)

				if err != nil {
					return err
				}

				files[dir][path] = info.ModTime()

				ch <- Change{
					Operation: "add",
					Base:      dir,
					Path:      rel,
				}
			}

			return nil
		})

		time.Sleep(700 * time.Millisecond)
	}
}
