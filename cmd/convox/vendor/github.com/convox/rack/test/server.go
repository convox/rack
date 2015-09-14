package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func Server(t *testing.T, stubs ...Http) *httptest.Server {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		found := false

		for _, stub := range stubs {
			if stub.Method == r.Method && stub.Path == r.URL.Path {
				data, err := json.Marshal(stub.Response)

				if err != nil {
					w.WriteHeader(503)
					w.Write(serverError(err.Error()))
				}

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
