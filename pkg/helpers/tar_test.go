//go:build unix

package helpers_test

import (
	"archive/tar"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/convox/rack/pkg/helpers"
)

func tarWithFiles(t *testing.T, n int) *bytes.Reader {
	t.Helper()

	var buf bytes.Buffer

	tw := tar.NewWriter(&buf)

	for i := 0; i < n; i++ {
		body := []byte(fmt.Sprintf("%d\n", i))

		h := &tar.Header{
			Name:     fmt.Sprintf("src/f%d", i),
			Typeflag: tar.TypeReg,
			Mode:     0644,
			Size:     int64(len(body)),
		}

		if err := tw.WriteHeader(h); err != nil {
			t.Fatalf("write header %d: %v", i, err)
		}

		if _, err := tw.Write(body); err != nil {
			t.Fatalf("write body %d: %v", i, err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}

	return bytes.NewReader(buf.Bytes())
}

func limitDescriptors(t *testing.T, soft uint64) {
	t.Helper()

	var orig syscall.Rlimit

	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &orig); err != nil {
		t.Skipf("getrlimit: %v", err)
	}

	if orig.Cur <= soft {
		t.Skipf("descriptor soft limit is already %d", orig.Cur)
	}

	t.Cleanup(func() {
		if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &orig); err != nil {
			t.Fatalf("restore descriptor limit: %v", err)
		}
	})

	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &syscall.Rlimit{Cur: soft, Max: orig.Max}); err != nil {
		t.Skipf("setrlimit: %v", err)
	}
}

func TestUnarchiveUnderLowDescriptorLimit(t *testing.T) {
	const files = 512

	limitDescriptors(t, 128)

	dir := t.TempDir()

	if err := helpers.Unarchive(tarWithFiles(t, files), dir); err != nil {
		t.Fatalf("unarchive: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(dir, "src"))
	if err != nil {
		t.Fatalf("read extracted directory: %v", err)
	}

	if len(entries) != files {
		t.Errorf("extracted %d files, want %d", len(entries), files)
	}

	last := filepath.Join(dir, "src", fmt.Sprintf("f%d", files-1))

	body, err := os.ReadFile(last)
	if err != nil {
		t.Fatalf("read %s: %v", last, err)
	}

	if want := fmt.Sprintf("%d\n", files-1); string(body) != want {
		t.Errorf("%s contains %q, want %q", last, string(body), want)
	}
}
