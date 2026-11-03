package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type Cache map[string]map[string]*CacheItem

type CacheItem struct {
	Item    interface{}
	Expires time.Time
}

var (
	cache       = Cache{}
	lock        = sync.Mutex{}
	generations = map[string]uint64{}
	flights     singleflight.Group
)

func Get(collection string, key interface{}) interface{} {
	lock.Lock()
	defer lock.Unlock()

	if os.Getenv("PROVIDER") == "test" {
		return nil
	}

	hash, err := hashKey(key)

	if err != nil {
		return nil
	}

	if cache[collection] == nil {
		return nil
	}

	item := cache[collection][hash]

	if item == nil {
		return nil
	}

	if item.Expires.Before(time.Now()) {
		return nil
	}

	return item.Item
}

func Set(collection string, key, value interface{}, ttl time.Duration) error {
	lock.Lock()
	defer lock.Unlock()

	if cache[collection] == nil {
		cache[collection] = map[string]*CacheItem{}
	}

	hash, err := hashKey(key)

	if err != nil {
		return err
	}

	cache[collection][hash] = &CacheItem{
		Item:    value,
		Expires: time.Now().Add(ttl),
	}

	return nil
}

func Clear(collection string, key interface{}) error {
	lock.Lock()
	defer lock.Unlock()

	generations[collection]++

	hash, err := hashKey(key)

	if err != nil {
		return err
	}

	if cache[collection] != nil && cache[collection][hash] != nil {
		delete(cache[collection], hash)
	}

	return nil
}

func ClearPrefix(collection string, prefix string) error {
	lock.Lock()
	defer lock.Unlock()

	generations[collection]++

	for k := range cache[collection] {
		match, err := hashPrefix(k, prefix)
		if err != nil {
			return err
		}

		if match {
			delete(cache[collection], k)
		}
	}

	return nil
}

// Fetch returns the cached value or runs fill once for all concurrent callers, storing its value when fill returns true.
func Fetch(collection string, key interface{}, ttl time.Duration, fill func() (interface{}, bool, error)) (interface{}, error) {
	if os.Getenv("PROVIDER") == "test" {
		v, _, err := fill()
		return v, err
	}

	hash, err := hashKey(key)
	if err != nil {
		return nil, err
	}

	lock.Lock()
	if item := cache[collection][hash]; item != nil && !item.Expires.Before(time.Now()) {
		lock.Unlock()
		return item.Item, nil
	}
	gen := generations[collection]
	lock.Unlock()

	v, err, _ := flights.Do(fmt.Sprintf("%s\x00%d\x00%s", collection, gen, hash), func() (interface{}, error) {
		v, store, err := fill()
		if err != nil || !store {
			return v, err
		}

		lock.Lock()
		defer lock.Unlock()

		if generations[collection] == gen {
			if cache[collection] == nil {
				cache[collection] = map[string]*CacheItem{}
			}

			cache[collection][hash] = &CacheItem{Item: v, Expires: time.Now().Add(ttl)}
		}

		return v, nil
	})

	return v, err
}

func hashKey(key interface{}) (string, error) {
	data, err := json.Marshal(key)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func hashPrefix(hash string, prefix string) (bool, error) {
	var w interface{}

	if err := json.Unmarshal([]byte(hash), &w); err != nil {
		return false, err
	}

	if s, ok := w.(string); ok {
		if strings.HasPrefix(s, prefix) {
			return true, nil
		}
	}

	return false, nil
}
