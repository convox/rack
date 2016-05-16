package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var files = map[string]map[string]time.Time{}

func main() {
	for _, watch := range os.Args {
		watchDirectory(watch)
	}

	// block forever
	select {}
}

func watchDirectory(dir string) error {
	abs, err := filepath.Abs(dir)

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

	go watchForChanges(sym)

	return nil
}

func watchForChanges(dir string) {
	for {
		for file, _ := range files[dir] {
			if _, err := os.Stat(file); os.IsNotExist(err) {
				rel, _ := filepath.Rel(dir, file)
				delete(files[dir], file)
				fmt.Printf("delete|%s|%s\n", dir, rel)
			}
		}

		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if path == dir {
				return nil
			}

			e, ok := files[dir][path]

			if !ok || e.Before(info.ModTime()) {
				rel, _ := filepath.Rel(dir, path)
				files[dir][path] = info.ModTime()
				fmt.Printf("add|%s|%s\n", dir, rel)
			}

			return nil
		})

		time.Sleep(900 * time.Millisecond)
	}
}
