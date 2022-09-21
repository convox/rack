package cache

import (
	"os"
	"strings"
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
