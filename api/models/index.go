package models

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type Index map[string]IndexItem

type IndexItem struct {
	Name    string      `json:"name"`
	Mode    os.FileMode `json:"mode"`
	ModTime time.Time   `json:"mtime"`
}

func IndexUpload(hash string, data []byte) error {
	bucket, err := indexBucket()

	if err != nil {
		return err
	}

	return S3Put(bucket, fmt.Sprintf("index/%s", hash), data, false)
}

func (index Index) Diff() ([]string, error) {
	missing := []string{}

	bucket, err := indexBucket()

	if err != nil {
		return nil, err
	}

	hashch := make(chan string)
	errch := make(chan error)

	for hash, _ := range index {
		go missingHash(bucket, hash, hashch, errch)
	}

	for range index {
		select {
		case hash := <-hashch:
			if hash != "" {
				missing = append(missing, hash)
			}
		case err := <-errch:
			return nil, err
		}
	}

	return missing, nil
}

func (index Index) Download(dir string) error {
	ch := make(chan error)

	bucket, err := indexBucket()

	if err != nil {
		return err
	}

	for hash, item := range index {
		go downloadItem(bucket, hash, item, dir, ch)
	}

	for range index {
		if err := <-ch; err != nil {
			return err
		}
	}

	return nil
}

func missingHash(bucket, hash string, hashch chan string, errch chan error) {
	exists, err := s3Exists(bucket, fmt.Sprintf("index/%s", hash))

	if err != nil {
		errch <- err
		return
	}

	if !exists {
		hashch <- hash
		return
	}

	hashch <- ""
}

func downloadItem(bucket, hash string, item IndexItem, dir string, ch chan error) {
	data, err := s3Get(bucket, fmt.Sprintf("index/%s", hash))

	if err != nil {
		ch <- err
		return
	}

	file := filepath.Join(dir, item.Name)

	err = os.MkdirAll(filepath.Dir(file), 0755)

	if err != nil {
		ch <- err
		return
	}

	err = ioutil.WriteFile(file, data, item.Mode)

	if err != nil {
		ch <- err
		return
	}

	err = os.Chtimes(file, item.ModTime, item.ModTime)

	if err != nil {
		ch <- err
		return
	}

	ch <- nil
}

func indexBucket() (string, error) {
	resources, err := ListResources(os.Getenv("RACK"))

	if err != nil {
		return "", err
	}

	bucket := resources["Settings"].Id

	if bucket == "" {
		return "", fmt.Errorf("invalid settings bucket")
	}

	return bucket, nil
}
