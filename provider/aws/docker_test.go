package aws_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/convox/rack/pkg/test/awsutil"
)

func stubDocker(cycles ...awsutil.Cycle) *httptest.Server {
	handler := awsutil.NewHandler(cycles)

	// Wrap the handler to transparently handle Docker version negotiation
	// requests that vary across Docker CLI versions. CI uses Docker 18.09.6
	// which sends /_ping and /v1.24/info before API calls, while newer
	// versions may skip /info or send different negotiation sequences.
	wrapper := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/_ping") {
			w.Header().Set("Api-Version", "1.24")
			w.Header().Set("Docker-Experimental", "false")
			w.Header().Set("Ostype", "linux")
			w.WriteHeader(200)
			w.Write([]byte("OK"))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/info") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write([]byte("{}"))
			return
		}
		handler.ServeHTTP(w, r)
	})

	s := httptest.NewServer(wrapper)

	os.Setenv("DOCKER_HOST", fmt.Sprintf("tcp://%s", s.URL[7:]))
	os.Setenv("DOCKER_API_VERSION", "1.24")

	return s
}

type fakeDocker struct {
	images []string
	fail   map[string]bool
	reject map[string]bool
	slow   map[string]time.Duration
	hold   map[string]chan struct{}

	mu     sync.Mutex
	calls  []string
	held   int
	loaded int64
}

func (f *fakeDocker) start(t *testing.T) {
	s := httptest.NewServer(f)

	t.Cleanup(func() {
		s.CloseClientConnections()
		s.Close()
	})

	t.Setenv("DOCKER_HOST", "tcp://"+s.Listener.Addr().String())
	t.Setenv("DOCKER_API_VERSION", "1.24")
	t.Setenv("DOCKER_CONFIG", t.TempDir())
}

func (f *fakeDocker) Calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string{}, f.calls...)
}

func (f *fakeDocker) Held() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.held
}

func (f *fakeDocker) Loaded() int64 {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.loaded
}

func (f *fakeDocker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1.24")
	q := r.URL.Query()

	var call string

	switch {
	case strings.HasSuffix(path, "/_ping"):
		w.Header().Set("Api-Version", "1.24")
		w.Header().Set("Ostype", "linux")
		_, _ = w.Write([]byte("OK"))
		return
	case path == "/info":
		_, _ = w.Write([]byte("{}"))
		return
	case path == "/auth":
		call = "login"
	case path == "/images/json":
		call = "images"
	case path == "/images/create":
		call = "pull " + q.Get("fromImage") + ":" + q.Get("tag")
	case path == "/images/get":
		call = "save"
	case path == "/images/load":
		call = "load"
	case r.Method == http.MethodDelete:
		call = "rmi " + strings.TrimPrefix(path, "/images/")
	case strings.HasSuffix(path, "/tag"):
		call = "tag " + q.Get("repo") + ":" + q.Get("tag")
	case strings.HasSuffix(path, "/push"):
		call = "push " + strings.TrimSuffix(strings.TrimPrefix(path, "/images/"), "/push") + ":" + q.Get("tag")
	default:
		call = r.Method + " " + path
	}

	if f.reject[call] {
		f.mu.Lock()
		f.calls = append(f.calls, call)
		f.mu.Unlock()

		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"rejected"}`))
		return
	}

	n, _ := io.Copy(io.Discard, r.Body)

	time.Sleep(f.slow[call])

	if ch := f.hold[strings.Fields(call)[0]]; ch != nil {
		f.mu.Lock()
		f.held++
		f.mu.Unlock()

		select {
		case <-ch:
		case <-r.Context().Done():
		}
	}

	f.mu.Lock()
	f.calls = append(f.calls, call)
	if call == "load" {
		f.loaded = n
	}
	f.mu.Unlock()

	if f.fail[call] || f.fail[strings.Fields(call)[0]] {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"failed"}`))
		return
	}

	switch call {
	case "images":
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{{"Id": "sha256:1", "RepoTags": f.images}})
	case "save":
		_, _ = w.Write([]byte("tar"))
	default:
		_, _ = w.Write([]byte("{}"))
	}
}
