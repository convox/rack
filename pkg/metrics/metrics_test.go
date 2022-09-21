package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPost(t *testing.T) {
	data := map[string]interface{}{}
	ht := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
	}))

	defer ht.Close()

	m := New(ht.URL)

	attrs := map[string]interface{}{
		"hello": "world",
	}
	err := m.Post("test", attrs)
	require.NoError(t, err)
	require.Equal(t, attrs, data)
}
