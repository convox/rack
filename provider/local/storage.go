package local

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/convox/rack/cache"
	"github.com/convox/rack/helpers"
)

var lock sync.Mutex

type BucketFunc func(bucket *bolt.Bucket) error

func (p *Provider) storageBucket(key string, fn BucketFunc) error {
	tx, err := p.db.Begin(true)
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

func (p *Provider) storageDelete(key string) error {
	path, name, err := storageKeyParts(key)
	if err != nil {
		return err
	}

	if err := cache.Clear("storage", key); err != nil {
		return err
	}

	return p.storageBucket(path, func(bucket *bolt.Bucket) error {
		return bucket.Delete([]byte(name))
	})
}

func (p *Provider) storageDeleteAll(prefix string) error {
	path, name, err := storageKeyParts(prefix)
	if err != nil {
		return err
	}

	if err := cache.ClearPrefix("storage", prefix); err != nil {
		return err
	}

	return p.storageBucket(path, func(bucket *bolt.Bucket) error {
		if bucket.Bucket([]byte(name)) == nil {
			return nil
		}
		return bucket.DeleteBucket([]byte(name))
	})
}

func (p *Provider) storageExists(key string) bool {
	path, name, err := storageKeyParts(key)
	if err != nil {
		return false
	}

	err = p.storageLoad(key, nil, 0)
	if err != nil {
		return false
	}

	err = p.storageBucket(path, func(bucket *bolt.Bucket) error {
		item := bucket.Get([]byte(name))
		if item == nil {
			return fmt.Errorf("not found")
		}
		return nil
	})

	return err == nil
}

func (p *Provider) storageList(prefix string) ([]string, error) {
	items := []string{}

	err := p.storageBucket(prefix, func(bucket *bolt.Bucket) error {
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

func (p *Provider) storageLoad(key string, v interface{}, d time.Duration) error {
	if w := cache.Get("storage", key); w != nil {
		if v != nil {
			reflect.ValueOf(v).Elem().Set(reflect.ValueOf(w).Elem())
		}
		return nil
	}

	data, err := p.storageRead(key)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	if err := cache.Set("storage", key, v, d); err != nil {
		return err
	}

	return nil
}

func (p *Provider) storageRead(key string) ([]byte, error) {
	path, name, err := storageKeyParts(key)
	if err != nil {
		return nil, err
	}

	var data []byte

	err = p.storageBucket(path, func(bucket *bolt.Bucket) error {
		data = bucket.Get([]byte(name))
		if data == nil {
			return fmt.Errorf("no such key: %s", key)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (p *Provider) storageStore(key string, v interface{}) error {
	path, name, err := storageKeyParts(key)
	if err != nil {
		return err
	}

	if err := cache.Clear("storage", key); err != nil {
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

	return p.storageBucket(path, func(bucket *bolt.Bucket) error {
		return bucket.Put([]byte(name), data)
	})
}

func (p *Provider) storageLogRead(key string, since time.Time, fn func(at time.Time, entry []byte)) error {
	return p.storageBucket(key, func(bucket *bolt.Bucket) error {
		return bucket.ForEach(func(k, v []byte) error {
			at, err := time.Parse(helpers.SortableTime, string(k))
			if err != nil {
				return err
			}
			if at.After(since) {
				fn(at, v)
			}
			return nil
		})
	})
}

func (p *Provider) storageLogWrite(key string, entry []byte) error {
	return p.storageBucket(key, func(bucket *bolt.Bucket) error {
		return bucket.Put([]byte(time.Now().Format(helpers.SortableTime)), entry)
	})
}

func storageKeyParts(key string) (string, string, error) {
	parts := strings.Split(key, "/")

	if len(parts) < 2 {
		return "", "", fmt.Errorf("cannot pop key: %s", key)
	}

	return strings.Join(parts[0:len(parts)-1], "/"), parts[len(parts)-1], nil
}
