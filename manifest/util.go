package manifest

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	ResizeChannel chan os.Signal
	TerminalWidth = 0
)

func coalesce(values ...string) string {
	for _, s := range values {
		if s != "" {
			return s
		}
	}

	return ""
}

func prefixReader(prefix string, r io.Reader) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		printWrap(prefix, scanner.Text())

	}
}

func printWrap(prefix string, text string) {
	if TerminalWidth == 0 {
		fmt.Printf("%s %s\n", prefix, text)
	} else {
		wrap := TerminalWidth - len(prefix) - 1
		lines := splitLines(text, wrap)
		fmt.Printf("%s %s\n", prefix, strings.Join(lines, "\n"+strings.Repeat(" ", len(prefix)-1)+"| "))
	}
}

func splitLines(text string, width int) []string {
	s := []string{}

	for {
		if len(text) > width {
			s = append(s, text[0:width])
			text = text[width:]
		} else {
			s = append(s, text)
			break
		}
	}

	return s
}
