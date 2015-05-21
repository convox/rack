package main

import (
	"flag"
	"fmt"
	"os"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "build: turn a convox application into an ami\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s <name> <repository> [ref]\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n  build example-sinatra https://github.com/convox-examples/sinatra.git master\n")
	}
}

func main() {
	push := flag.String("push", "", "push build to this prefix when done")
	token := flag.String("token", os.Getenv("GITHUB_TOKEN"), "github access token")

	flag.Parse()

	l := len(flag.Args())
	if l < 2 || l > 3 {
		flag.Usage()
		os.Exit(0)
	}

	args := flag.Args()

	builder := NewBuilder()
	builder.GitHubToken = *token

	app := positional(args, 0)
	repo := positional(args, 1)
	ref := positional(args, 2)

	err := builder.Build(repo, app, ref, *push)

	if err != nil {
		fmt.Printf("error|%s\n", err)
		os.Exit(1)
	}
}

func positional(args []string, n int) string {
	if len(args) > n {
		return args[n]
	} else {
		return ""
	}
}
