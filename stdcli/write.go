package stdcli

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
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
			"fail":   RenderAttributes(203),
			"header": RenderAttributes(242),
			"ok":     RenderAttributes(46),
			"start":  RenderAttributes(253),
			"wait":   RenderAttributes(228),
		},
	}
}

func Error(err error) error {
	return DefaultWriter.Error(err)
}

func Errorf(format string, args ...interface{}) error {
	return DefaultWriter.Errorf(format, args...)
}

func OK() (int, error) {
	return DefaultWriter.OK()
}

func Sprintf(format string, args ...interface{}) string {
	return DefaultWriter.Sprintf(format, args...)
}

func Startf(format string, args ...interface{}) (int, error) {
	return DefaultWriter.Startf(format, args...)
}

type tagWriter struct {
	io.Writer
	tag string
}

func (w tagWriter) Write(data []byte) (int, error) {
	cparts := []string{}

	for _, line := range strings.Split(string(data), "\n") {
		cparts = append(cparts, fmt.Sprintf("<%s>%s</%s>", w.tag, line, w.tag))
	}

	cdata := strings.Join(cparts, "\n")

	if _, err := w.Writer.Write([]byte(Sprintf(cdata))); err != nil {
		return 0, err
	}

	return len(data), nil
}

func TagWriter(tag string, w io.Writer) io.Writer {
	return tagWriter{Writer: w, tag: tag}
}

func Wait(status string) (int, error) {
	return DefaultWriter.Wait(status)
}

func Write(data []byte) (int, error) {
	return DefaultWriter.Write(data)
}

func Writef(format string, args ...interface{}) (int, error) {
	return DefaultWriter.Writef(format, args...)
}

type errorFormatter struct {
	error
	w *Writer
}

func (err errorFormatter) Format(s fmt.State, verb rune) {
	fmt.Fprintf(s, err.w.renderTags("<error>%s</error>"), err.error)
}

func (w *Writer) Error(err error) error {
	return errorFormatter{error: err, w: w}
}

func (w *Writer) Errorf(format string, args ...interface{}) error {
	return w.Error(fmt.Errorf(format, args...))
}

func (w *Writer) OK() (int, error) {
	return w.Writef("<ok>OK</ok>\n")
}

func (w *Writer) Sprintf(format string, args ...interface{}) string {
	return fmt.Sprintf(w.renderTags(format), args...)
}

func (w *Writer) Startf(format string, args ...interface{}) (int, error) {
	finish := ""

	if os.Getenv("CONVOX_DEBUG") == "true" {
		finish = "\n"
	}

	return w.Writef("<start>%s</start><start>:</start> %s", w.Sprintf(format, args...), finish)
}

func (w *Writer) Wait(status string) (int, error) {
	return w.Writef("<wait>%s</wait>\n", status)
}

func (w *Writer) Write(data []byte) (int, error) {
	return w.Stdout.Write(data)
}

func (w *Writer) Writef(format string, args ...interface{}) (int, error) {
	return w.Write([]byte(fmt.Sprintf(w.renderTags(format), args...)))
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

func RenderAttributes(attrs ...int) Renderer {
	return func(s string) string {
		s = stripTag(s)
		for _, a := range attrs {
			s = fmt.Sprintf("\033[38;5;%dm", a) + s
		}
		return s + "\033[0m"
	}
}

func renderError(s string) string {
	return fmt.Sprintf("\033[38;5;196mERROR: \033[38;5;203m%s\033[0m", stripTag(s))
}

var colorStripper = regexp.MustCompile("\033\\[[^m]+m")

func stripColor(s string) string {
	return colorStripper.ReplaceAllString(s, "")
}

var tagStripper = regexp.MustCompile(`<[^>?]+>(.*?)</[^>?]+>`)

func stripTag(s string) string {
	match := tagStripper.FindStringSubmatch(s)

	if len(match) != 2 {
		panic(fmt.Sprintf("could not strip tags: %s", s))
	}

	return match[1]
}
