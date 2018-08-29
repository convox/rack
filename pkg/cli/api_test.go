package cli_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestApi(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("Get", "/apps", stdsdk.RequestOptions{}, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			err := json.Unmarshal([]byte(`[{"name":"app1"}]`), args.Get(2))
			require.NoError(t, err)
		})

		res, err := testExecute(e, "api get /apps", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, "[\n  {\n    \"name\": \"app1\"\n  }\n]\n", res.Stdout)
		require.Equal(t, "", res.Stderr)
	})
}

func TestApiError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("Get", "/apps", stdsdk.RequestOptions{}, mock.Anything).Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "api get /apps", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		require.Equal(t, "", res.Stdout)
		require.Equal(t, "ERROR: err1\n", res.Stderr)
	})
}
