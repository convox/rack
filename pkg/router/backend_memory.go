package router

import (
	"sync"
	"time"
)

type BackendMemory struct {
	activity sync.Map
	active   sync.Map
	idle     sync.Map
	routes   sync.Map

	activityLock sync.Mutex
	targetLock   sync.Mutex
	// activity map[string]time.Time
	// active   map[string]int
	// idle     map[string]bool
	// routes   map[string]map[string]bool
}

func NewBackendMemory() *BackendMemory {
	return &BackendMemory{
		activity: sync.Map{},
		active:   sync.Map{},
		idle:     sync.Map{},
		routes:   sync.Map{},
	}
}

func (b *BackendMemory) ActivityBegin(host string) error {
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

func (b *BackendMemory) ActivityEnd(host string) error {
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

func (b *BackendMemory) ActivityLatest(host string) (time.Time, error) {
	v, ok := b.activity.Load(host)
	if !ok {
		return time.Time{}, nil
	}

	t, ok := v.(time.Time)
	if !ok {
		return time.Time{}, nil
	}

	return t, nil
}

func (b *BackendMemory) activeCount(host string) (int64, error) {
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

func (b *BackendMemory) IdleGet(host string) (bool, error) {
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

func (b *BackendMemory) IdleReady(cutoff time.Time) ([]string, error) {
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

func (b *BackendMemory) IdleSet(host string, idle bool) error {
	b.idle.Store(host, idle)

	return nil
}

// func (b *BackendMemory) Route(host string) (string, error) {
//   return "", nil
// }

func (b *BackendMemory) TargetAdd(host, target string) error {
	b.targetLock.Lock()
	defer b.targetLock.Unlock()

	ts := b.targets(host)

	ts[target] = true

	b.routes.Store(host, ts)

	return nil
}

func (b *BackendMemory) TargetList(host string) ([]string, error) {
	b.targetLock.Lock()
	defer b.targetLock.Unlock()

	ts := b.targets(host)

	targets := []string{}

	for t := range ts {
		targets = append(targets, t)
	}

	return targets, nil
}

func (b *BackendMemory) TargetRemove(host, target string) error {
	b.targetLock.Lock()
	defer b.targetLock.Unlock()

	ts := b.targets(host)

	delete(ts, target)

	b.routes.Store(host, ts)

	return nil
}

func (b *BackendMemory) targets(host string) map[string]bool {
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
