package helpers_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/stretchr/testify/require"
)

func TestRetry(t *testing.T) {
	testData := []struct {
		errUntil  int
		expectErr bool
	}{
		{
			errUntil:  8,
			expectErr: false,
		},
		{
			errUntil:  0,
			expectErr: false,
		},
		{
			errUntil:  30,
			expectErr: true,
		},
	}

	for _, td := range testData {
		cnt := 0
		err := helpers.Retry(10, 1*time.Millisecond, func() error {
			if cnt >= td.errUntil {
				return nil
			}
			cnt++
			return fmt.Errorf("error")
		})
		if td.expectErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
}
