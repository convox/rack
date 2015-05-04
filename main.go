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
	region := flag.String("region", "us-east-1", "aws region")
	access := flag.String("access", os.Getenv("AWS_ACCESS"), "aws access id")
	secret := flag.String("secret", os.Getenv("AWS_SECRET"), "aws secret key")
	token := flag.String("token", os.Getenv("GITHUB_ACCESS_TOKEN"), "github access token")
	public := flag.Bool("public", false, "make ami public")

	flag.Parse()

	if len(flag.Args()) != 2 {
		flag.Usage()
		os.Exit(0)
	}

	args := flag.Args()

	builder := NewBuilder()
	builder.AwsRegion = *region
	builder.AwsAccess = *access
	builder.AwsSecret = *secret
	builder.GitHubToken = *token

	app := positional(args, 0)
	repo := positional(args, 1)
	ref := positional(args, 2)

	err := builder.Build(repo, app, ref, *public)

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
