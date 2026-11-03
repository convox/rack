package cache

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSet(t *testing.T) {
	err := Set("TestSet", "test", map[string]interface{}{
		"string": "empty",
		"int":    1,
	}, time.Minute)
	require.NoError(t, err)
}

func TestGet(t *testing.T) {
	if os.Getenv("PROVIDER") == "test" {
		os.Setenv("PROVIDER", "")
		defer os.Setenv("PROVIDER", "test")
	}

	value := map[string]interface{}{
		"string": "empty",
		"int":    1,
	}

	err := Set("TestGet", "test1", value, time.Minute)
	require.NoError(t, err)

	data := Get("TestGet", "test1")
	require.NotNil(t, data)
	require.Equal(t, value, data)

	data = Get("TestGet", "test2")
	require.Nil(t, data)
}

func TestGetExpiredData(t *testing.T) {
	if os.Getenv("PROVIDER") == "test" {
		os.Setenv("PROVIDER", "")
		defer os.Setenv("PROVIDER", "test")
	}

	value := map[string]interface{}{
		"string": "empty",
		"int":    1,
	}
	err := Set("TestGetExpiredData", "test1", value, time.Second*2)
	require.NoError(t, err)

	data := Get("TestGetExpiredData", "test1")
	require.NotNil(t, data)
	require.Equal(t, value, data)

	time.Sleep(3 * time.Second)
	data = Get("TestGetExpiredData", "test1")
	require.Nil(t, data)
}

func TestClear(t *testing.T) {
	if os.Getenv("PROVIDER") == "test" {
		os.Setenv("PROVIDER", "")
		defer os.Setenv("PROVIDER", "test")
	}

	value := map[string]interface{}{
		"string": "empty",
		"int":    1,
	}

	err := Set("TestClear", "test1", value, time.Minute)
	require.NoError(t, err)

	data := Get("TestClear", "test1")
	require.NotNil(t, data)
	require.Equal(t, value, data)

	err = Clear("TestClear", "test1")
	require.NoError(t, err)

	data = Get("TestClear", "test1")
	require.Nil(t, data)
}

func TestClearPrefix(t *testing.T) {
	if os.Getenv("PROVIDER") == "test" {
		os.Setenv("PROVIDER", "")
		defer os.Setenv("PROVIDER", "test")
	}

	value := map[string]interface{}{
		"string": "empty",
		"int":    1,
	}

	keys := []string{
		"test1", "test2", "test_345", "notest",
	}
	for _, k := range keys {
		err := Set("TestClearPrefix", k, value, time.Minute)
		require.NoError(t, err)

		data := Get("TestClearPrefix", k)
		require.NotNil(t, data)
	}

	prefix := "te"
	err := ClearPrefix("TestClearPrefix", prefix)
	require.NoError(t, err)

	for _, k := range keys {
		data := Get("TestClearPrefix", k)
		if strings.HasPrefix(k, prefix) {
			require.Nil(t, data)
		} else {
			require.NotNil(t, data)
		}
	}
}

func noProviderTest(t *testing.T) {
	t.Setenv("PROVIDER", "")

	require.NoError(t, Clear(t.Name(), "k"))
	t.Cleanup(func() { require.NoError(t, Clear(t.Name(), "k")) })
}

func TestFetchCoalesces(t *testing.T) {
	noProviderTest(t)

	var calls int32
	release := make(chan struct{})
	started := make(chan struct{})

	fill := func() (interface{}, bool, error) {
		if atomic.AddInt32(&calls, 1) == 1 {
			close(started)
		}
		<-release
		return nil, true, fmt.Errorf("fill failed")
	}

	errs := make(chan error, 10)

	go func() {
		_, err := Fetch(t.Name(), "k", time.Minute, fill)
		errs <- err
	}()

	<-started

	var entered sync.WaitGroup
	entered.Add(9)

	for i := 0; i < 9; i++ {
		go func() {
			entered.Done()
			_, err := Fetch(t.Name(), "k", time.Minute, fill)
			errs <- err
		}()
	}

	entered.Wait()
	time.Sleep(200 * time.Millisecond)
	close(release)

	for i := 0; i < 10; i++ {
		require.EqualError(t, <-errs, "fill failed")
	}

	require.EqualValues(t, 1, atomic.LoadInt32(&calls))
	require.Nil(t, Get(t.Name(), "k"))
}

func TestFetchClearDuringFill(t *testing.T) {
	noProviderTest(t)

	release := make(chan struct{})
	started := make(chan struct{})
	done := make(chan interface{})

	go func() {
		v, err := Fetch(t.Name(), "k", time.Minute, func() (interface{}, bool, error) {
			close(started)
			<-release
			return "old", true, nil
		})
		if err != nil {
			t.Error(err)
		}
		done <- v
	}()

	<-started
	require.NoError(t, Clear(t.Name(), "k"))

	var calls int32
	v, err := Fetch(t.Name(), "k", time.Minute, func() (interface{}, bool, error) {
		atomic.AddInt32(&calls, 1)
		return "new", true, nil
	})
	require.NoError(t, err)
	require.Equal(t, "new", v)
	require.EqualValues(t, 1, calls)

	close(release)
	require.Equal(t, "old", <-done)
	require.Equal(t, "new", Get(t.Name(), "k"))
}

func TestFetchPanicReleasesKey(t *testing.T) {
	noProviderTest(t)

	require.Panics(t, func() {
		_, _ = Fetch(t.Name(), "k", time.Minute, func() (interface{}, bool, error) {
			panic("boom")
		})
	})

	v, err := Fetch(t.Name(), "k", time.Minute, func() (interface{}, bool, error) {
		return "ok", true, nil
	})
	require.NoError(t, err)
	require.Equal(t, "ok", v)
}

func TestFetchStoreFlag(t *testing.T) {
	noProviderTest(t)

	var calls int32
	fill := func(store bool) func() (interface{}, bool, error) {
		return func() (interface{}, bool, error) {
			atomic.AddInt32(&calls, 1)
			return "v", store, nil
		}
	}

	_, err := Fetch(t.Name(), "k", time.Minute, fill(false))
	require.NoError(t, err)
	require.Nil(t, Get(t.Name(), "k"))

	_, err = Fetch(t.Name(), "k", time.Minute, fill(true))
	require.NoError(t, err)

	v, err := Fetch(t.Name(), "k", time.Minute, fill(true))
	require.NoError(t, err)
	require.Equal(t, "v", v)
	require.EqualValues(t, 2, calls)
}

func TestFetchProviderTest(t *testing.T) {
	t.Setenv("PROVIDER", "test")

	var calls int32
	for i := 0; i < 2; i++ {
		v, err := Fetch("TestFetchProviderTest", "k", time.Minute, func() (interface{}, bool, error) {
			atomic.AddInt32(&calls, 1)
			return "v", true, nil
		})
		require.NoError(t, err)
		require.Equal(t, "v", v)
	}

	require.EqualValues(t, 2, calls)
}
