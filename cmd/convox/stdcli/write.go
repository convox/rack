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
			"fail":   RenderAttributes(203),
			"header": RenderAttributes(242),
			"ok":     RenderAttributes(46),
			"start":  RenderAttributes(247),
			"wait":   RenderAttributes(228),
			"warn":   RenderAttributes(208),
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

// Warn prints a string in dark orange.
func Warn(msg string) (int, error) {
	return DefaultWriter.Warn(fmt.Sprintf("WARNING: %s", msg))
}

func Sprintf(format string, args ...interface{}) string {
	return DefaultWriter.Sprintf(format, args...)
}

func Startf(format string, args ...interface{}) (int, error) {
	return DefaultWriter.Startf(format, args...)
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

func (w *Writer) Error(err error) error {
	err = ErrorStdCli(err.Error())
	if err.Error() != "Token expired" {
		w.Stderr.Write([]byte(fmt.Sprintf(w.renderTags("<error>%s</error>\n"), err)))
	}
	return err
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
	return w.Writef("<start>%s</start><start>...</start> ", w.Sprintf(format, args...))
}

func (w *Writer) Wait(status string) (int, error) {
	return w.Writef("<wait>%s</wait>\n", status)
}

// Warn wraps a string in <warn> tags
func (w *Writer) Warn(status string) (int, error) {
	return w.Writef("<warn>%s</warn>\n", status)
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
	return fmt.Sprintf("\033[38;5;124mERROR: \033[38;5;203m%s\033[0m", stripTag(s))
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
