package storage

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/boltdb/bolt"
)

type Storage struct {
	db *bolt.DB
}

type bucketFunc func(bucket *bolt.Bucket) error

func Open(path string) (*Storage, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	s := &Storage{db: db}

	return s, nil
}

func (s *Storage) bucket(key string, fn bucketFunc) error {
	tx, err := s.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	cur := tx.Bucket([]byte("rack"))

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

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) List(prefix string) ([]string, error) {
	items := []string{}

	err := s.bucket(prefix, func(bucket *bolt.Bucket) error {
		return bucket.ForEach(func(k, v []byte) error {
			items = append(items, string(k))
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return items, nil
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

func (s *Storage) Read(key string) ([]byte, error) {
	path, name, err := storageKeyParts(key)
	if err != nil {
		return nil, err
	}

	var dcp []byte

	err = s.bucket(path, func(bucket *bolt.Bucket) error {
		data := bucket.Get([]byte(name))
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

func storageKeyParts(key string) (string, string, error) {
	parts := strings.Split(key, "/")

	if len(parts) < 2 {
		return "", "", fmt.Errorf("cannot pop key: %s", key)
	}

	return strings.Join(parts[0:len(parts)-1], "/"), parts[len(parts)-1], nil
}
