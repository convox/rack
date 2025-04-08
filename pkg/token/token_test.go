package token_test

import (
	"encoding/json"
	"testing"

	"github.com/convox/rack/pkg/token"
	"github.com/stretchr/testify/require"
)

func TestAuthenticateError(t *testing.T) {
	req := map[string]interface{}{
		"publicKey": map[string]interface{}{
			"challenge": "Y2hlbGxlbmdl",
			"timeout":   120,
			"rpId":      "rpId1",
			"allowCredentials": []map[string]interface{}{
				{
					"type": "type1",
					"id":   "id1",
				},
			},
			"userVerification": "error",
		},
	}

	reqBytes, err := json.Marshal(req)
	require.NoError(t, err)

	_, err = token.Authenticate(reqBytes)
	require.Error(t, err)
}
