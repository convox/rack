package logstore

import (
	"context"
	"sort"
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
	subs    map[chan Log]subscription
}

type Stream struct {
	Group string
	Name  string
	Logs  []log
	lock  sync.Mutex
	subs  map[chan Log]subscription
}

type Log struct {
	Group     string
	Stream    string
	Timestamp time.Time
	Message   string
}

type subscription struct {
	context context.Context
	after   time.Time
}

type Subscribe func(context.Context, chan Log, time.Time, bool)

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
		subs:    map[chan Log]subscription{},
	}
}

func newStream(group, name string) *Stream {
	return &Stream{
		Group: group,
		Name:  name,
		Logs:  []log{},
		subs:  map[chan Log]subscription{},
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

	g.lock.Lock()
	defer g.lock.Unlock()

	for ch, sub := range g.subs {
		select {
		case <-sub.context.Done():
			continue
		default:
			if ts.After(sub.after) {
				ch <- Log{Group: g.Name, Stream: stream, Timestamp: ts, Message: message}
			}
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

func (g *Group) Subscribe(ctx context.Context, ch chan Log, since time.Time, follow bool) {
	g.lock.Lock()
	defer g.lock.Unlock()

	logs := []Log{}

	for _, s := range g.Streams {
		for _, l := range s.Logs {
			if l.Timestamp.After(since) {
				logs = append(logs, Log{Group: g.Name, Stream: s.Name, Timestamp: l.Timestamp, Message: l.Message})
			}
		}
	}

	latest := stream(ch, logs)

	if follow {
		g.subs[ch] = subscription{context: ctx, after: latest}
		go g.watchSubscription(ctx, ch)
	} else {
		close(ch)
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

func (g *Group) watchSubscription(ctx context.Context, ch chan Log) {
	<-ctx.Done()
	g.lock.Lock()
	defer g.lock.Unlock()
	delete(g.subs, ch)
	close(ch)
}

func (s *Stream) Append(ts time.Time, message string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.Logs = append(s.Logs, log{Timestamp: ts, Message: message})

	for ch, sub := range s.subs {
		if ts.After(sub.after) {
			select {
			case <-sub.context.Done():
				continue
			default:
				ch <- Log{Group: s.Group, Stream: s.Name, Timestamp: ts, Message: message}
			}
		}
	}
}

func (s *Stream) Subscribe(ctx context.Context, ch chan Log, since time.Time, follow bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	logs := []Log{}

	for _, l := range s.Logs {
		if l.Timestamp.After(since) {
			logs = append(logs, Log{Group: s.Group, Stream: s.Name, Timestamp: l.Timestamp, Message: l.Message})
		}
	}

	latest := stream(ch, logs)

	if follow {
		s.subs[ch] = subscription{context: ctx, after: latest}
		go s.watchSubscription(ctx, ch)
	} else {
		close(ch)
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

func (s *Stream) watchSubscription(ctx context.Context, ch chan Log) {
	<-ctx.Done()
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.subs, ch)
	close(ch)
}

func stream(ch chan Log, logs []Log) time.Time {
	latest := time.Time{}

	sort.Slice(logs, func(i, j int) bool { return logs[i].Timestamp.Before(logs[j].Timestamp) })

	for _, l := range logs {
		ch <- l
	}

	if len(logs) > 1 {
		latest = logs[len(logs)-1].Timestamp
	}

	return latest
}
