package aws_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/convox/rack/pkg/test/awsutil"
)

func stubDocker(cycles ...awsutil.Cycle) *httptest.Server {
	handler := awsutil.NewHandler(cycles)

	// Wrap the handler to transparently handle Docker ping requests
	// that vary across Docker CLI versions (GET vs HEAD, with or without
	// version prefix). This prevents cycle mismatches from version
	// negotiation differences.
	wrapper := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/_ping") {
			w.Header().Set("Api-Version", "1.24")
			w.Header().Set("Docker-Experimental", "false")
			w.Header().Set("Ostype", "linux")
			w.WriteHeader(200)
			w.Write([]byte("OK"))
			return
		}
		handler.ServeHTTP(w, r)
	})

	s := httptest.NewServer(wrapper)

	os.Setenv("DOCKER_HOST", fmt.Sprintf("tcp://%s", s.URL[7:]))
	os.Setenv("DOCKER_API_VERSION", "1.24")

	return s
}
