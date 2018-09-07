package start

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"

	"github.com/convox/rack/pkg/prefix"
)

type Interface interface {
	Start1(context.Context, Options1) error
	Start2(context.Context, Options2) error
}

type Options struct {
	App      string
	Build    bool
	Cache    bool
	Manifest string
	Output   io.Writer
	Sync     bool

	writer *prefix.Writer
}

type Start struct{}

func (o Options) Writef(prefix, format string, args ...interface{}) {
	if o.writer == nil {
		return
	}

	o.writer.Writef(prefix, format, args...)
}

func (o Options) Writer(prefix string) io.Writer {
	if o.writer == nil {
		return ioutil.Discard
	}

	return o.writer.Writer(prefix)
}

func (o Options) prefixWriter(services map[string]bool) *prefix.Writer {
	if o.Output == nil {
		return nil
	}

	prefixes := map[string]string{
		"build":  "system",
		"convox": "system",
	}

	for s := range services {
		prefixes[s] = fmt.Sprintf("color%d", prefixHash(s))
	}

	return prefix.NewWriter(o.Output, prefixes)
}

func prefixHash(prefix string) int {
	sum := 0

	for c := range prefix {
		sum += int(c)
	}

	return sum % 18
}

func handleInterrupt(fn func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)
	<-ch
	fmt.Println("")
	fn()
}
