package manifest

import (
	"fmt"
	"sync"

	"github.com/fatih/color"
)

var StaticColors = map[string]color.Attribute{
	"convox": color.FgWhite,
	"build":  color.FgWhite,
}

var PrefixColors = []color.Attribute{
	color.FgMagenta,
	color.FgBlue,
	color.FgCyan,
	color.FgYellow,
	color.FgGreen,
}

type Stream chan string

type Output struct {
	colors   map[string]color.Attribute
	lock     sync.Mutex
	prefixes map[Stream]string
	streams  map[string]Stream
}

func NewOutput() Output {
	return Output{
		colors:   make(map[string]color.Attribute),
		prefixes: make(map[Stream]string),
		streams:  make(map[string]Stream),
	}
}

func (o *Output) Stream(prefix string) Stream {
	if s, ok := o.streams[prefix]; ok {
		return s
	}

	s := make(Stream)

	if color, ok := StaticColors[prefix]; ok {
		o.colors[prefix] = color
	} else {
		o.colors[prefix] = PrefixColors[len(o.prefixes)%len(PrefixColors)]
	}

	o.prefixes[s] = prefix
	o.streams[prefix] = s

	go o.watchStream(s)

	return s
}

func (o *Output) paddedPrefix(s Stream) string {
	color := color.New(o.colors[o.prefixes[s]]).Add(color.Bold)
	color.EnableColor()

	return fmt.Sprintf(color.SprintfFunc()("%%-%ds │", o.widestPrefix()), o.prefixes[s])
}

func (o *Output) printLine(s Stream, line string) {
	o.lock.Lock()
	defer o.lock.Unlock()

	fmt.Printf("%s %s\n", o.paddedPrefix(s), line)
}

func (o *Output) watchStream(s Stream) {
	for line := range s {
		o.printLine(s, line)
	}
}

func (o *Output) widestPrefix() (w int) {
	for _, p := range o.prefixes {
		if len(p) > w {
			w = len(p)
		}
	}

	return
}
