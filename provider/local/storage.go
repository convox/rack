package local

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/boltdb/bolt"
)

const (
	RootBucket = "root"
)

type BucketFunc func(*bolt.Bucket) error

type Storage struct {
	File string

	db *bolt.DB
}

func NewStorage(file string) (*Storage, error) {
	s := &Storage{File: file}

	db, err := bolt.Open(file, 0600, nil)
	if err != nil {
		return nil, err
	}

	tx, err := db.Begin(true)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.CreateBucketIfNotExists([]byte(RootBucket)); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.db = db

	return s, nil
}

func (s *Storage) Clear(prefix string) error {
	path, name, err := splitKey(prefix)
	if err != nil {
		return err
	}

	return s.bucket(path, func(b *bolt.Bucket) error {
		if b.Bucket([]byte(name)) == nil {
			return nil
		}
		return b.DeleteBucket([]byte(name))
	})
}

func (s *Storage) Load(key string, v interface{}) error {
	data, err := s.Read(key)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	return nil
}

func (s *Storage) List(prefix string) ([]string, error) {
	items := []string{}

	err := s.bucket(prefix, func(b *bolt.Bucket) error {
		return b.ForEach(func(k, v []byte) error {
			items = append(items, string(k))
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (s *Storage) Read(key string) ([]byte, error) {
	path, name, err := splitKey(key)
	if err != nil {
		return nil, err
	}

	var dcp []byte

	err = s.bucket(path, func(b *bolt.Bucket) error {
		data := b.Get([]byte(name))
		if data == nil {
			return fmt.Errorf("no such key: %s", key)
		}
		dcp = make([]byte, len(data))
		copy(dcp, data)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return dcp, nil
}

func (s *Storage) Write(key string, v interface{}) error {
	path, name, err := splitKey(key)
	if err != nil {
		return err
	}

	var data []byte

	switch t := v.(type) {
	case io.Reader:
		data, err = ioutil.ReadAll(t)
	default:
		data, err = json.Marshal(v)
	}
	if err != nil {
		return err
	}

	return s.bucket(path, func(b *bolt.Bucket) error {
		return b.Put([]byte(name), data)
	})
}

func (s *Storage) bucket(key string, fn BucketFunc) error {
	tx, err := s.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	cur := tx.Bucket([]byte(RootBucket))

	for _, kp := range strings.Split(key, "/") {
		b, err := cur.CreateBucketIfNotExists([]byte(kp))
		if err != nil {
			return err
		}
		cur = b
	}

	if err := fn(cur); err != nil {
		return err
	}

	return tx.Commit()
}

func splitKey(key string) (string, string, error) {
	parts := strings.Split(key, "/")

	if len(parts) < 2 {
		return "", "", fmt.Errorf("cannot pop key: %s", key)
	}

	return strings.Join(parts[0:len(parts)-1], "/"), parts[len(parts)-1], nil
}
