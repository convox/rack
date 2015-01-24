package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <repo> <app> [ref]\n", os.Args[0])
		flag.PrintDefaults()
	}

	region := flag.String("region", "us-east-1", "aws region")
	access := flag.String("access", os.Getenv("AWS_ACCESS"), "aws access id")
	secret := flag.String("secret", os.Getenv("AWS_SECRET"), "aws secret key")

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

	repo := positional(args, 0)
	app := positional(args, 1)
	ref := positional(args, 2)

	err := builder.Build(repo, app, ref)

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
