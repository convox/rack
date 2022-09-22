package helpers_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/stretchr/testify/require"
)

func TestWait(t *testing.T) {
	testData := []struct {
		errUntil   int
		times      int
		timoutMili int
		expectErr  bool
	}{
		{
			errUntil:   8,
			times:      10,
			timoutMili: 150,
			expectErr:  false,
		},
		{
			errUntil:   0,
			times:      10,
			timoutMili: 150,
			expectErr:  false,
		},
		{
			errUntil:   30,
			times:      10,
			timoutMili: 150,
			expectErr:  true,
		},
		{
			errUntil:   10,
			times:      10,
			timoutMili: 1,
			expectErr:  true,
		},
	}

	for _, td := range testData {
		cnt := 0
		err := helpers.Wait(1*time.Millisecond, time.Duration(td.timoutMili)*time.Millisecond, td.times, func() (bool, error) {
			if cnt >= td.errUntil {
				return true, nil
			}
			cnt++
			return false, fmt.Errorf("error")
		})
		if td.expectErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
}
