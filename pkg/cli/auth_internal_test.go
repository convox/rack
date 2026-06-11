package cli

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestU2fCallbackServerDecodesData(t *testing.T) {
	dataChan := make(chan []byte, 1)
	errChan := make(chan error, 1)
	addr, srv, err := u2fCallbackServer(dataChan, errChan)
	require.NoError(t, err)
	defer srv.Close()

	payload := []byte(`{"id":"abc","response":{"signature":"x"}}`)
	encoded := base64.StdEncoding.EncodeToString(payload)

	res, err := http.Get(addr + "/?data=" + url.QueryEscape(encoded))
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)

	select {
	case got := <-dataChan:
		require.Equal(t, payload, got)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for callback data")
	}
}

func TestU2fCallbackServerHandlesPlusInBase64(t *testing.T) {
	dataChan := make(chan []byte, 1)
	errChan := make(chan error, 1)
	addr, srv, err := u2fCallbackServer(dataChan, errChan)
	require.NoError(t, err)
	defer srv.Close()

	payload := append([]byte{0xfb, 0xff, 0xfe}, []byte(`{"id":"abc"}`)...)
	encoded := base64.StdEncoding.EncodeToString(payload)
	require.Contains(t, encoded, "+", "test payload must produce a '+' in standard base64")

	// the console redirects to ...?data=<base64> without url-escaping, so the '+' arrives raw
	res, err := http.Get(addr + "/?data=" + encoded)
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)

	select {
	case got := <-dataChan:
		require.Equal(t, payload, got)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for callback data")
	}
}

func TestU2fCallbackServerSecondCallbackDoesNotBlock(t *testing.T) {
	dataChan := make(chan []byte, 1)
	errChan := make(chan error, 1)
	addr, srv, err := u2fCallbackServer(dataChan, errChan)
	require.NoError(t, err)
	defer srv.Close()

	get := func(payload string) int {
		enc := base64.StdEncoding.EncodeToString([]byte(payload))
		res, err := http.Get(addr + "/?data=" + url.QueryEscape(enc))
		require.NoError(t, err)
		defer res.Body.Close()
		return res.StatusCode
	}

	require.Equal(t, http.StatusOK, get("first"))
	require.Equal(t, http.StatusOK, get("second"))

	select {
	case got := <-dataChan:
		require.Contains(t, [][]byte{[]byte("first"), []byte("second")}, got)
	case <-time.After(2 * time.Second):
		t.Fatal("expected a buffered callback value")
	}

	require.Len(t, errChan, 0)
}

func TestU2fCallbackServerRejectsMissingData(t *testing.T) {
	dataChan := make(chan []byte, 1)
	errChan := make(chan error, 1)
	addr, srv, err := u2fCallbackServer(dataChan, errChan)
	require.NoError(t, err)
	defer srv.Close()

	res, err := http.Get(addr + "/")
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusBadRequest, res.StatusCode)
}
