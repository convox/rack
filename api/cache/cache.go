package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)
import "github.com/convox/rack/Godeps/_workspace/src/github.com/convox/logger"

type Cache map[string]map[string]*CacheItem

type CacheItem struct {
	Item    interface{}
	Expires time.Time
}

var (
	cache = Cache{}
	lock  = sync.Mutex{}
	log   = logger.New("ns=rack.api.cache")
)

func Get(collection string, key interface{}) interface{} {
	hash, err := hashKey(key)

	if err != nil {
		log.Log("fn=get collection=%q key=%q status=error error=%q", collection, hash, err)
		return nil
	}

	if cache[collection] == nil {
		log.Log("fn=get collection=%q key=%q status=miss", collection, hash)
		return nil
	}

	item := cache[collection][hash]

	if item == nil {
		log.Log("fn=get collection=%q key=%q status=miss", collection, hash)
		return nil
	}

	if item.Expires.Before(time.Now()) {
		log.Log("fn=get collection=%q key=%q status=expired", collection, hash)
		return nil
	}

	log.Log("fn=get collection=%q key=%q status=hit", collection, hash)
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
		log.Log("fn=set collection=%q key=%q status=error error=%q", collection, hash, err)
		return err
	}

	cache[collection][hash] = &CacheItem{
		Item:    value,
		Expires: time.Now().Add(ttl),
	}

	log.Log("fn=set collection=%q key=%q status=success", collection, hash)
	return nil
}

func Clear(collection string, key interface{}) error {
	lock.Lock()
	defer lock.Unlock()

	hash, err := hashKey(key)

	if err != nil {
		log.Log("fn=clearcollection=%q key=%q status=error error=%q", collection, hash, err)
		return err
	}

	if cache[collection] != nil && cache[collection][hash] != nil {
		delete(cache[collection], hash)
	}

	return nil
}

func hashKey(key interface{}) (string, error) {
	data, err := json.Marshal(key)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", sha256.Sum256(data))[0:32], nil
}
