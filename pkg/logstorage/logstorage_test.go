package logstorage_test

import (
	"context"
	"testing"
	"time"

	"github.com/convox/rack/pkg/logstorage"
	"github.com/stretchr/testify/require"
)

var (
	time1 = time.Date(2019, 01, 01, 0, 1, 0, 0, time.UTC)
	time2 = time.Date(2019, 01, 01, 0, 2, 0, 0, time.UTC)
	time3 = time.Date(2019, 01, 01, 0, 3, 0, 0, time.UTC)
)

func TestNoFollow(t *testing.T) {
	s := logstorage.New()

	s.Append("foo", time2, "p2", "two")
	s.Append("foo", time1, "p1", "one")
	s.Append("foo", time3, "p3", "three")

	ch := make(chan logstorage.Log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.Subscribe(ctx, ch, "foo", time1, false)

	log, ok := <-ch
	require.True(t, ok)
	require.Equal(t, "p1", log.Prefix)
	require.Equal(t, "one", log.Message)
	require.Equal(t, time1, log.Timestamp)

	log, ok = <-ch
	require.True(t, ok)
	require.Equal(t, "p2", log.Prefix)
	require.Equal(t, "two", log.Message)
	require.Equal(t, time2, log.Timestamp)

	log, ok = <-ch
	require.True(t, ok)
	require.Equal(t, "p3", log.Prefix)
	require.Equal(t, "three", log.Message)
	require.Equal(t, time3, log.Timestamp)

	_, ok = <-ch
	require.False(t, ok)
}

func TestFollow(t *testing.T) {
	s := logstorage.New()

	s.Append("foo", time2, "p2", "two")

	ch := make(chan logstorage.Log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.Subscribe(ctx, ch, "foo", time1, true)

	time.Sleep(500 * time.Millisecond)

	s.Append("foo", time3, "p3", "three")
	s.Append("foo", time1, "p1", "one")

	log, ok := <-ch
	require.True(t, ok)
	require.Equal(t, "two", log.Message)
	require.Equal(t, "p2", log.Prefix)
	require.Equal(t, time2, log.Timestamp)

	log, ok = <-ch
	require.True(t, ok)
	require.Equal(t, time3, log.Timestamp)
	require.Equal(t, "p3", log.Prefix)
	require.Equal(t, "three", log.Message)

	log, ok = <-ch
	require.True(t, ok)
	require.Equal(t, time1, log.Timestamp)
	require.Equal(t, "p1", log.Prefix)
	require.Equal(t, "one", log.Message)
}
