package logstorage

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

type Store struct {
	lock          sync.Mutex
	streams       map[string][]Log
	subscriptions subscriptions
}

type Log struct {
	Prefix    string
	Message   string
	Timestamp time.Time
}

type Receiver chan Log

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func New() Store {
	s := Store{streams: map[string][]Log{}}

	go s.startCleaner()

	return s
}

func (s *Store) Append(stream string, ts time.Time, prefix, message string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	log := Log{Message: message, Prefix: prefix, Timestamp: ts}

	ls, ok := s.streams[stream]
	if !ok {
		ls = []Log{}
	}

	n := sort.Search(len(ls), func(i int) bool { return ls[i].Timestamp.After(ts) })

	ls = append(ls, Log{})
	copy(ls[n+1:], ls[n:])
	ls[n] = log

	s.streams[stream] = ls

	s.subscriptions.send(stream, log)
}

func (s *Store) Subscribe(ctx context.Context, ch Receiver, stream string, start time.Time, follow bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if ls, ok := s.streams[stream]; ok {
		n := sort.Search(len(ls), func(i int) bool { return !ls[i].Timestamp.Before(start) })
		go sendMultiple(ch, ls[n:], func() {
			if !follow {
				close(ch)
			}
		})
	}

	if follow {
		s.subscriptions.Subscribe(ctx, ch, stream, start)
	}
}

func (s *Store) cleanupLogs() {
	s.lock.Lock()
	defer s.lock.Unlock()

	for name := range s.streams {
		ls := s.streams[name]
		n := sort.Search(len(ls), func(i int) bool { return ls[i].Timestamp.After(time.Now().Add(30 * time.Second)) })
		s.streams[name] = ls[n:]
	}
}

func (s *Store) startCleaner() {
	for range time.Tick(30 * time.Second) {
		s.cleanupLogs()
	}
}

type subscriptions struct {
	lock          sync.Mutex
	subscriptions map[string]map[string]*subscription
}

type subscription struct {
	ch    Receiver
	lock  sync.Mutex
	queue []Log
	start time.Time
}

func (s *subscriptions) Subscribe(ctx context.Context, ch Receiver, stream string, start time.Time) {
	s.add(ctx, ch, stream, start)
}

func (s *subscriptions) add(ctx context.Context, ch Receiver, stream string, start time.Time) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.subscriptions == nil {
		s.subscriptions = map[string]map[string]*subscription{}
	}

	if _, ok := s.subscriptions[stream]; !ok {
		s.subscriptions[stream] = map[string]*subscription{}
	}

	handle := fmt.Sprintf("%v:%d", ch, rand.Int63())

	s.subscriptions[stream][handle] = &subscription{ch: ch, start: start}

	go s.watch(ctx, stream, handle)
}

func (s *subscriptions) remove(stream, handle string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.subscriptions[stream]; !ok {
		return
	}

	delete(s.subscriptions[stream], handle)
}

func (s *subscriptions) send(stream string, l Log) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.subscriptions[stream]; !ok {
		return
	}

	for _, sub := range s.subscriptions[stream] {
		if !sub.start.After(l.Timestamp) {
			sub.add(l)
		}
	}
}

func (s *subscriptions) watch(ctx context.Context, stream, handle string) {
	defer s.remove(stream, handle)

	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if ss, ok := s.subscriptions[stream]; ok {
				if sub, ok := ss[handle]; ok && len(sub.queue) > 0 {
					sub.flush()
				}
			}
		}
	}
}

func (s *subscription) add(l Log) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.queue = append(s.queue, l)
}

func (s *subscription) flush() {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, l := range s.queue {
		s.ch <- l
	}

	s.queue = s.queue[:0]
}

func sendMultiple(ch Receiver, ls []Log, done func()) {
	defer done()
	for _, l := range ls {
		ch <- l
	}
}
