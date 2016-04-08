package aws

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/convox/rack/api/cache"
	"github.com/convox/rack/api/structs"
)

var (
	IndexOperationConcurrency = 128
)

func (p *AWSProvider) IndexDiff(index *structs.Index) ([]string, error) {
	missing := []string{}

	bucket := os.Getenv("SETTINGS_BUCKET")

	inch := make(chan string)
	outch := make(chan string)
	errch := make(chan error)

	for i := 1; i < IndexOperationConcurrency; i++ {
		go p.missingHashes(bucket, inch, outch, errch)
	}

	go func() {
		for hash := range *index {
			inch <- hash
		}
	}()

	for range *index {
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

func (p *AWSProvider) IndexDownload(index *structs.Index, dir string) error {
	bucket := os.Getenv("SETTINGS_BUCKET")

	inch := make(chan string)
	errch := make(chan error)

	for i := 1; i < IndexOperationConcurrency; i++ {
		go p.downloadItems(bucket, *index, dir, inch, errch)
	}

	go func() {
		for hash := range *index {
			inch <- hash
		}
	}()

	for range *index {
		if err := <-errch; err != nil {
			return err
		}
	}

	return nil
}

func (p *AWSProvider) IndexUpload(hash string, data []byte) error {
	return p.s3Put(os.Getenv("SETTINGS_BUCKET"), fmt.Sprintf("index/%s", hash), data, false)
}

func (p *AWSProvider) downloadItems(bucket string, index structs.Index, dir string, inch chan string, errch chan error) {
	for hash := range inch {
		errch <- p.downloadItem(bucket, hash, index[hash], dir)
	}
}

func (p *AWSProvider) downloadItem(bucket, hash string, item structs.IndexItem, dir string) error {
	data, err := p.s3Get(bucket, fmt.Sprintf("index/%s", hash))

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

func (p *AWSProvider) missingHashes(bucket string, inch, outch chan string, errch chan error) {
	for hash := range inch {
		exists, err := p.hashExists(bucket, hash)

		if err != nil {
			errch <- err
		} else if !exists {
			outch <- hash
		} else {
			outch <- ""
		}
	}
}

func (p *AWSProvider) hashExists(bucket, hash string) (bool, error) {
	if exists, ok := cache.Get("index.missingHash", hash).(bool); ok && exists {
		return true, nil
	}

	exists, err := p.s3Exists(bucket, fmt.Sprintf("index/%s", hash))

	if err != nil {
		return false, err
	}

	if exists {
		cache.Set("index.missingHash", hash, true, 30*24*time.Hour)
	}

	return exists, nil
}
