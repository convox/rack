package stdcli

import (
	"fmt"
	"io"
	"os"
	"regexp"
)

var (
	DefaultWriter *Writer
)

type Renderer func(string) string

type Writer struct {
	Color  bool
	Stdout io.Writer
	Stderr io.Writer
	Tags   map[string]Renderer
}

func init() {
	DefaultWriter = &Writer{
		Color:  IsTerminal(os.Stdout),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tags: map[string]Renderer{
			"error":  renderError,
			"header": RenderColors(242),
			"h1":     RenderColors(244),
			"h2":     RenderColors(241),
			"id":     RenderColors(247),
			"info":   RenderColors(247),
			"ok":     RenderColors(46),
			"start":  RenderColors(247),
			"u":      RenderUnderline(),
			"value":  RenderColors(251),
		},
	}
}

func (w *Writer) Error(err error) error {
	fmt.Fprintf(w.Stderr, w.renderTags("<error>%s</error>\n"), err)
	return err
}

func (w *Writer) Errorf(format string, args ...interface{}) error {
	return w.Error(fmt.Errorf(format, args...))
}

func (w *Writer) IsTerminal() bool {
	if f, ok := w.Stdout.(*os.File); ok {
		return IsTerminal(f)
	}

	return false
}

func (w *Writer) Sprintf(format string, args ...interface{}) string {
	return fmt.Sprintf(w.renderTags(format), args...)
}

func (w *Writer) Write(data []byte) (int, error) {
	return w.Stdout.Write([]byte(w.renderTags(string(data))))
}

func (w *Writer) Writef(format string, args ...interface{}) (int, error) {
	return fmt.Fprintf(w.Stdout, w.renderTags(format), args...)
}

func (w *Writer) renderTags(s string) string {
	for tag, render := range w.Tags {
		s = regexp.MustCompile(fmt.Sprintf("<%s>(.*?)</%s>", tag, tag)).ReplaceAllStringFunc(s, render)
	}

	if !w.Color {
		s = stripColor(s)
	}

	return s
}

func RenderColors(colors ...int) Renderer {
	return func(s string) string {
		s = stripTag(s)
		for _, c := range colors {
			s = fmt.Sprintf("\033[38;5;%dm", c) + s
		}
		return s + "\033[0m"
	}
}

func RenderUnderline() Renderer {
	return func(s string) string {
		return fmt.Sprintf("\033[4m%s\033[24m", stripTag(s))
	}
}

func renderError(s string) string {
	return fmt.Sprintf("\033[38;5;124mERROR: \033[38;5;203m%s\033[0m", stripTag(s))
}

var (
	colorStripper = regexp.MustCompile("\033\\[[^m]+m")
	tagStripper   = regexp.MustCompile(`^<[^>?]+>(.*)</[^>?]+>$`)
)

func stripColor(s string) string {
	return colorStripper.ReplaceAllString(s, "")
}

func stripTag(s string) string {
	match := tagStripper.FindStringSubmatch(s)

	if len(match) != 2 {
		return s
	}

	return match[1]
}
