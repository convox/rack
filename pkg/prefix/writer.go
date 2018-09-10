package prefix

import (
	"bufio"
	"fmt"
	"io"
	"sync"
)

type Writer struct {
	lock     sync.Mutex
	max      int
	prefixes map[string]string
	writer   io.Writer
}

func NewWriter(w io.Writer, prefixes map[string]string) Writer {
	max := 0

	for k := range prefixes {
		if l := len(k); l > max {
			max = l
		}
	}

	return Writer{max: max, prefixes: prefixes, writer: w}
}

func (w Writer) Write(prefix string, r io.Reader) {
	s := bufio.NewScanner(r)

	for s.Scan() {
		w.Writef(prefix, "%s\n", s.Text())
	}
}

func (w Writer) Writer(prefix string) io.Writer {
	rr, ww := io.Pipe()

	go w.Write(prefix, rr)

	return ww
}

func (w Writer) Writef(prefix string, format string, args ...interface{}) {
	w.lock.Lock()
	defer w.lock.Unlock()

	line := fmt.Sprintf(w.format(prefix), prefix, fmt.Sprintf(format, args...))

	fmt.Fprintf(w.writer, line)
}

func (w Writer) format(prefix string) string {
	ot := ""
	ct := ""

	if t := w.prefixes[prefix]; t != "" {
		ot = fmt.Sprintf("<%s>", t)
		ct = fmt.Sprintf("</%s>", t)
	}

	return fmt.Sprintf("%s%%-%ds%s | %%s", ot, w.max, ct)
}
