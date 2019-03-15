package test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Server(t *testing.T, stubs ...Http) *httptest.Server {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		found := false

		for _, stub := range stubs {
			headersMatch := true
			for k, v := range stub.Headers {
				if r.Header.Get(k) != v {
					headersMatch = false
					break
				}
			}

			if stub.Method == r.Method && stub.Path == r.URL.Path && headersMatch {
				data, err := json.Marshal(stub.Response)

				if err != nil {
					w.WriteHeader(503)
					w.Write(serverError(err.Error()))
				}

				rb, err := ioutil.ReadAll(r.Body)

				if err != nil {
					w.WriteHeader(503)
					w.Write(serverError(err.Error()))
				}

				assert.Equal(t, stub.Body, string(rb))

				w.WriteHeader(stub.Code)
				w.Write(data)

				found = true
				break
			}
		}

		if !found {
			fmt.Fprintf(os.Stderr, "Missing HTTP stub:\n")
			fmt.Fprintf(os.Stderr, "  %s %s\n", r.Method, r.URL.Path)
			t.Fail()

			w.WriteHeader(404)
			w.Write(serverError("stub not found"))
		}
	}))

	return ts
}

func serverError(message string) []byte {
	return []byte(fmt.Sprintf(`{"error":%q}`, message))
}
