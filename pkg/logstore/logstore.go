package logstore

import (
	"sync"
	"time"
)

const (
	logCleanupTick     = 5 * time.Minute
	logStorageDuration = 1 * time.Hour
)

type Store struct {
	Groups map[string]*Group
	lock   sync.Mutex
}

type Group struct {
	Name    string
	Streams map[string]*Stream
	lock    sync.Mutex
	subs    map[chan Log]time.Time
}

type Stream struct {
	Group string
	Name  string
	Logs  []log
	lock  sync.Mutex
	subs  map[chan Log]time.Time
}

type Log struct {
	Group     string
	Stream    string
	Timestamp time.Time
	Message   string
}

type Subscribe func(chan Log, time.Time) func()

type log struct {
	Timestamp time.Time
	Message   string
}

func New() *Store {
	s := &Store{Groups: map[string]*Group{}}

	go s.cleanupTicker()

	return s
}

func newGroup(name string) *Group {
	return &Group{
		Name:    name,
		Streams: map[string]*Stream{},
		subs:    map[chan Log]time.Time{},
	}
}

func newStream(group, name string) *Stream {
	return &Stream{
		Group: group,
		Name:  name,
		Logs:  []log{},
	}
}

func (s *Store) Group(name string) *Group {
	s.lock.Lock()
	defer s.lock.Unlock()

	if g, ok := s.Groups[name]; ok {
		return g
	}

	g := newGroup(name)

	s.Groups[name] = g

	return g
}

func (s *Store) Append(group, stream string, ts time.Time, message string) {
	s.Group(group).Append(stream, ts, message)
}

func (s *Store) cleanupTicker() {
	for range time.Tick(logCleanupTick) {
		s.cleanup()
	}
}

func (s *Store) cleanup() {
	s.lock.Lock()
	defer s.lock.Unlock()

	for name, group := range s.Groups {
		group.cleanup()

		if len(group.Streams) == 0 && len(group.subs) == 0 {
			delete(s.Groups, name)
		}
	}
}

func (g *Group) Append(stream string, ts time.Time, message string) {
	g.Stream(stream).Append(ts, message)

	for sub, since := range g.subs {
		if ts.After(since) {
			sub <- Log{Group: g.Name, Stream: stream, Timestamp: ts, Message: message}
		}
	}
}

func (g *Group) Stream(name string) *Stream {
	g.lock.Lock()
	defer g.lock.Unlock()

	if s, ok := g.Streams[name]; ok {
		return s
	}

	s := newStream(g.Name, name)

	g.Streams[name] = s

	return s
}

func (g *Group) Subscribe(ch chan Log, since time.Time) func() {
	g.lock.Lock()
	defer g.lock.Unlock()

	latest := time.Time{}

	for _, s := range g.Streams {
		if l := s.stream(ch, since); l.After(latest) {
			latest = l
		}
	}

	g.subs[ch] = latest

	return func() {
		g.lock.Lock()
		defer g.lock.Unlock()
		delete(g.subs, ch)
	}
}

func (g *Group) cleanup() {
	g.lock.Lock()
	defer g.lock.Unlock()

	for name, stream := range g.Streams {
		stream.cleanup()

		if len(stream.Logs) == 0 && len(stream.subs) == 0 {
			delete(g.Streams, name)
		}
	}
}

func (s *Stream) Append(ts time.Time, message string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.Logs = append(s.Logs, log{Timestamp: ts, Message: message})

	for sub, since := range s.subs {
		if ts.After(since) {
			sub <- Log{Group: s.Group, Stream: s.Name, Timestamp: ts, Message: message}
		}
	}
}

func (s *Stream) Subscribe(ch chan Log, since time.Time) func() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.subs[ch] = s.stream(ch, since)

	return func() {
		s.lock.Lock()
		defer s.lock.Unlock()
		delete(s.subs, ch)
	}
}

func (s *Stream) cleanup() {
	s.lock.Lock()
	defer s.lock.Unlock()

	for i, l := range s.Logs {
		if l.Timestamp.After(time.Now().UTC().Add(-1 * logStorageDuration)) {
			s.Logs = s.Logs[i:]
			return
		}
	}

	s.Logs = []log{}
}

func (s *Stream) stream(ch chan Log, since time.Time) time.Time {
	latest := time.Time{}

	for _, l := range s.Logs {
		ch <- Log{Group: s.Group, Stream: s.Name, Timestamp: l.Timestamp, Message: l.Message}
		if l.Timestamp.After(latest) {
			latest = l.Timestamp
		}
	}

	return latest
}
