package cache

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"
)

type Cache map[string]map[string]*CacheItem

type CacheItem struct {
	Item    interface{}
	Expires time.Time
}

var (
	cache = Cache{}
	lock  = sync.Mutex{}
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
