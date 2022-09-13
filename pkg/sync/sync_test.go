package sync_test

import (
	"testing"

	"github.com/convox/rack/pkg/sync"
	"github.com/stretchr/testify/require"
)

func TestContains(t *testing.T) {
	syncObj, err := sync.NewSync("container", "local", "remote", []string{})
	require.NoError(t, err)

	testData := []struct {
		newSync sync.Sync
		expect  bool
	}{
		{
			newSync: sync.Sync{
				Local:  syncObj.Local + "/test/1",
				Remote: syncObj.Remote + "/test/1",
			},
			expect: true,
		},
		{
			newSync: sync.Sync{
				Local:  syncObj.Local + "/test/1",
				Remote: syncObj.Remote + "/test/0",
			},
			expect: false,
		},
	}

	for _, td := range testData {
		require.Equal(t, td.expect, syncObj.Contains(td.newSync))
	}
}
