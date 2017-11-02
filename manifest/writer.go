package manifest

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/convox/rack/stdcli"
)

type PrefixWriter struct {
	Writer func(string) error
	buffer bytes.Buffer
}

var writeLock sync.Mutex

func (m *Manifest) WriteLine(line string) {
	writeLock.Lock()
	defer writeLock.Unlock()
	fmt.Println(line)
}

func (m *Manifest) Writef(label string, format string, args ...interface{}) {
	m.Writer(label, stdcli.DefaultWriter).Write([]byte(fmt.Sprintf(format, args...)))
}

var lock sync.Mutex

func init() {
	// color := rand.Intn(256)
	// for i := 0; i < 32; i++ {
	//   stdcli.DefaultWriter.Tags[fmt.Sprintf("color%d", i)] = stdcli.RenderAttributes(color)
	//   color += 15
	//   color %= 256
	// }
	for i := 0; i < 18; i++ {
		stdcli.DefaultWriter.Tags[fmt.Sprintf("color%d", i)] = stdcli.RenderAttributes(237 + i)
	}

	stdcli.DefaultWriter.Tags["dir"] = stdcli.RenderAttributes(246)
	stdcli.DefaultWriter.Tags["name"] = stdcli.RenderAttributes(246)
}

func (m *Manifest) Writer(label string, w io.Writer) *PrefixWriter {
	hash := 0
	for _, c := range label {
		hash += int(c)
	}
	color := hash % 18

	prefix := []byte(stdcli.Sprintf(fmt.Sprintf("<color%d>%%-%ds</color%d> | ", color, m.prefixLength(), color), label))

	return &PrefixWriter{
		Writer: func(s string) error {
			lock.Lock()
			defer lock.Unlock()

			if _, err := w.Write(prefix); err != nil {
				return err
			}

			if _, err := w.Write([]byte(stdcli.DefaultWriter.Sprintf(s))); err != nil {
				return err
			}

			return nil
		},
	}
}

func (w *PrefixWriter) Write(p []byte) (int, error) {
	q := bytes.Replace(p, []byte{10, 13}, []byte{10}, -1)

	if _, err := w.buffer.Write(q); err != nil {
		return 0, err
	}

	for {
		idx := bytes.Index(w.buffer.Bytes(), []byte{10})
		if idx == -1 {
			break
		}

		if err := w.Writer(string(w.buffer.Next(idx + 1))); err != nil {
			return 0, err
		}
	}

	return len(p), nil
}

func (w PrefixWriter) Writef(format string, args ...interface{}) error {
	_, err := w.Write([]byte(fmt.Sprintf(format, args...)))
	return err
}

func (m *Manifest) prefixLength() int {
	max := 7 // "release"

	for _, s := range m.Services {
		if len(s.Name) > max {
			max = len(s.Name)
		}
	}

	return max
}
