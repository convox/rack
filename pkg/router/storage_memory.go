package router

import (
	"fmt"
	"sync"
	"time"
)

type StorageMemory struct {
	activity activityTracker
	idle     sync.Map
	idles    sync.Map
	routes   sync.Map

	targetLock sync.Mutex
}

func NewStorageMemory() *StorageMemory {
	fmt.Printf("ns=storage.memory at=new\n")

	return &StorageMemory{
		idle:   sync.Map{},
		routes: sync.Map{},
	}
}

func (s *StorageMemory) IdleGet(target string) (bool, error) {
	fmt.Printf("ns=storage.memory at=idle.get target=%q\n", target)

	v, ok := s.idle.Load(target)
	if !ok {
		return false, nil
	}

	i, ok := v.(bool)
	if !ok {
		return false, nil
	}

	return i, nil
}

func (s *StorageMemory) IdleSet(target string, idle bool) error {
	fmt.Printf("ns=storage.memory at=idle.get target=%q idle=%t\n", target, idle)

	s.idle.Store(target, idle)

	return nil
}

func (s *StorageMemory) RequestBegin(target string) error {
	fmt.Printf("ns=storage.memory at=request.begin target=%q\n", target)

	if err := s.activity.Begin(target); err != nil {
		return err
	}

	return nil
}

func (s *StorageMemory) RequestEnd(target string) error {
	fmt.Printf("ns=storage.memory at=request.end target=%q\n", target)

	if err := s.activity.End(target); err != nil {
		return err
	}

	return nil
}

func (s *StorageMemory) Stale(cutoff time.Time) ([]string, error) {
	fmt.Printf("ns=storage.memory at=stale cutoff=%s\n", cutoff)

	tsh := map[string]bool{}

	s.routes.Range(func(k, v interface{}) bool {
		host, ok := k.(string)
		if !ok {
			return true
		}

		for t := range s.targets(host) {
			tsh[t] = true
		}

		return true
	})

	stale := []string{}

	for t := range tsh {
		if v, ok := s.idles.Load(t); !ok || !v.(bool) {
			continue
		}

		a, err := s.activity.ActiveSince(t, cutoff)
		if err != nil {
			return nil, err
		}

		if !a {
			stale = append(stale, t)
		}
	}

	return stale, nil
}

func (s *StorageMemory) TargetAdd(host, target string, idles bool) error {
	fmt.Printf("ns=storage.memory at=target.add host=%q target=%q idles=%t\n", host, target, idles)

	s.targetLock.Lock()
	defer s.targetLock.Unlock()

	ts := s.targets(host)

	ts[target] = true

	s.activity.KeepAlive(target)
	s.idles.Store(target, idles)

	s.routes.Store(host, ts)

	return nil
}

func (s *StorageMemory) TargetList(host string) ([]string, error) {
	s.targetLock.Lock()
	defer s.targetLock.Unlock()

	ts := s.targets(host)

	targets := []string{}

	for t := range ts {
		targets = append(targets, t)
	}

	return targets, nil
}

func (s *StorageMemory) TargetRemove(host, target string) error {
	fmt.Printf("ns=storage.memory at=target.remove host=%q target=%q\n", host, target)

	s.targetLock.Lock()
	defer s.targetLock.Unlock()

	ts := s.targets(host)

	delete(ts, target)

	s.routes.Store(host, ts)

	return nil
}

func (s *StorageMemory) targets(host string) map[string]bool {
	v, ok := s.routes.Load(host)
	if !ok {
		return map[string]bool{}
	}

	h, ok := v.(map[string]bool)
	if !ok {
		return map[string]bool{}
	}

	return h
}

type activityTracker struct {
	activity  sync.Map
	counts    sync.Map
	countLock sync.Mutex
}

func (t *activityTracker) ActiveSince(key string, cutoff time.Time) (bool, error) {
	a, err := t.Activity(key)
	if err != nil {
		return false, err
	}

	c, err := t.Count(key)
	if err != nil {
		return false, err
	}

	return a.After(cutoff) || c > 0, nil
}

func (t *activityTracker) Activity(key string) (time.Time, error) {
	av, _ := t.activity.LoadOrStore(key, time.Time{})

	if a, ok := av.(time.Time); ok {
		return a, nil
	}

	return time.Time{}, fmt.Errorf("invalid activity type: %T", av)
}

func (t *activityTracker) Begin(key string) error {
	t.activity.Store(key, time.Now().UTC())

	if err := t.addCount(key, 1); err != nil {
		return err
	}

	return nil
}

func (t *activityTracker) Count(key string) (int64, error) {
	t.countLock.Lock()
	defer t.countLock.Unlock()

	cv, _ := t.counts.LoadOrStore(key, int64(0))

	if c, ok := cv.(int64); ok {
		return c, nil
	}

	return 0, fmt.Errorf("invalid count type: %T", cv)
}

func (t *activityTracker) End(key string) error {
	return t.addCount(key, -1)
}

func (t *activityTracker) KeepAlive(key string) error {
	t.activity.Store(key, time.Now().UTC())

	return nil
}

func (t *activityTracker) addCount(key string, n int64) error {
	t.countLock.Lock()
	defer t.countLock.Unlock()

	cv, _ := t.counts.LoadOrStore(key, int64(0))

	c, ok := cv.(int64)
	if !ok {
		return fmt.Errorf("invalid count type: %T", cv)
	}

	t.counts.Store(key, c+n)

	return nil
}
