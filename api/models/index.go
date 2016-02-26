package models

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/convox/rack/api/cache"
)

var (
	IndexOperationConcurrency = 128
)

type Index map[string]IndexItem

type IndexItem struct {
	Name    string      `json:"name"`
	Mode    os.FileMode `json:"mode"`
	ModTime time.Time   `json:"mtime"`
}

func IndexUpload(hash string, data []byte) error {
	return S3Put(os.Getenv("SETTINGS_BUCKET"), fmt.Sprintf("index/%s", hash), data, false)
}

func (index Index) Diff() ([]string, error) {
	missing := []string{}

	bucket := os.Getenv("SETTINGS_BUCKET")

	inch := make(chan string)
	outch := make(chan string)
	errch := make(chan error)

	for i := 1; i < IndexOperationConcurrency; i++ {
		go missingHashes(bucket, inch, outch, errch)
	}

	go func() {
		for hash := range index {
			inch <- hash
		}
	}()

	for range index {
		select {
		case hash := <-outch:
			if hash != "" {
				missing = append(missing, hash)
			}
		case err := <-errch:
			return nil, err
		}
	}

	close(inch)

	return missing, nil
}

func (index Index) Download(dir string) error {
	bucket := os.Getenv("SETTINGS_BUCKET")

	inch := make(chan string)
	errch := make(chan error)

	for i := 1; i < IndexOperationConcurrency; i++ {
		go downloadItems(bucket, index, dir, inch, errch)
	}

	go func() {
		for hash := range index {
			inch <- hash
		}
	}()

	for range index {
		if err := <-errch; err != nil {
			return err
		}
	}

	return nil
}

func missingHashes(bucket string, inch, outch chan string, errch chan error) {
	for hash := range inch {
		exists, err := hashExists(bucket, hash)

		if err != nil {
			errch <- err
		} else if !exists {
			outch <- hash
		} else {
			outch <- ""
		}
	}
}

func hashExists(bucket, hash string) (bool, error) {
	if exists, ok := cache.Get("index.missingHash", hash).(bool); ok && exists {
		return true, nil
	}

	exists, err := s3Exists(bucket, fmt.Sprintf("index/%s", hash))

	if err != nil {
		return false, err
	}

	if exists {
		cache.Set("index.missingHash", hash, true, 30*24*time.Hour)
	}

	return exists, nil
}

func downloadItems(bucket string, index Index, dir string, inch chan string, errch chan error) {
	for hash := range inch {
		errch <- downloadItem(bucket, hash, index[hash], dir)
	}
}

func downloadItem(bucket, hash string, item IndexItem, dir string) error {
	data, err := s3Get(bucket, fmt.Sprintf("index/%s", hash))

	if err != nil {
		return err
	}

	file := filepath.Join(dir, item.Name)

	err = os.MkdirAll(filepath.Dir(file), 0755)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(file, data, item.Mode)

	if err != nil {
		return err
	}

	err = os.Chtimes(file, item.ModTime, item.ModTime)

	if err != nil {
		return err
	}

	return nil
}
