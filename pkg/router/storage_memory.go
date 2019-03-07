package router

import (
	"sync"
	"time"
)

type StorageMemory struct {
	activity sync.Map
	active   sync.Map
	idle     sync.Map
	routes   sync.Map

	activityLock sync.Mutex
	targetLock   sync.Mutex
}

func NewStorageMemory() *StorageMemory {
	return &StorageMemory{
		activity: sync.Map{},
		active:   sync.Map{},
		idle:     sync.Map{},
		routes:   sync.Map{},
	}
}

func (b *StorageMemory) IdleGet(host string) (bool, error) {
	v, ok := b.idle.Load(host)
	if !ok {
		return false, nil
	}

	i, ok := v.(bool)
	if !ok {
		return false, nil
	}

	return i, nil
}

func (b *StorageMemory) IdleReady(cutoff time.Time) ([]string, error) {
	hosts := []string{}

	b.activity.Range(func(k, v interface{}) bool {
		h, ok := k.(string)
		if !ok {
			return true
		}

		t, ok := v.(time.Time)
		if !ok {
			return true
		}

		if t.Before(cutoff) {
			idle, err := b.IdleGet(h)
			if err != nil {
				return true
			}

			if !idle {
				hosts = append(hosts, h)
			}
		}

		return true
	})

	return hosts, nil
}

func (b *StorageMemory) IdleSet(host string, idle bool) error {
	b.idle.Store(host, idle)

	return nil
}

func (b *StorageMemory) RequestBegin(host string) error {
	b.activityLock.Lock()
	defer b.activityLock.Unlock()

	c, err := b.activeCount(host)
	if err != nil {
		return err
	}

	b.activity.Store(host, time.Now().UTC())
	b.active.Store(host, c+1)

	return nil
}

func (b *StorageMemory) RequestEnd(host string) error {
	b.activityLock.Lock()
	defer b.activityLock.Unlock()

	c, err := b.activeCount(host)
	if err != nil {
		return err
	}

	if c < 1 {
		return nil
	}

	b.activity.Store(host, time.Now().UTC())
	b.active.Store(host, c-1)

	return nil
}

func (b *StorageMemory) TargetAdd(host, target string) error {
	b.targetLock.Lock()
	defer b.targetLock.Unlock()

	ts := b.targets(host)

	ts[target] = true

	b.routes.Store(host, ts)

	return nil
}

func (b *StorageMemory) TargetList(host string) ([]string, error) {
	b.targetLock.Lock()
	defer b.targetLock.Unlock()

	ts := b.targets(host)

	targets := []string{}

	for t := range ts {
		targets = append(targets, t)
	}

	return targets, nil
}

func (b *StorageMemory) TargetRemove(host, target string) error {
	b.targetLock.Lock()
	defer b.targetLock.Unlock()

	ts := b.targets(host)

	delete(ts, target)

	b.routes.Store(host, ts)

	return nil
}

func (b *StorageMemory) activeCount(host string) (int64, error) {
	v, ok := b.active.Load(host)
	if !ok {
		return 0, nil
	}

	i, ok := v.(int64)
	if !ok {
		return 0, nil
	}

	return i, nil
}

func (b *StorageMemory) targets(host string) map[string]bool {
	v, ok := b.routes.Load(host)
	if !ok {
		return map[string]bool{}
	}

	h, ok := v.(map[string]bool)
	if !ok {
		return map[string]bool{}
	}

	return h
}
