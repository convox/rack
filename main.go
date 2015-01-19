package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <url> <cluster> <app>\n", os.Args[0])
		flag.PrintDefaults()
	}

	region := flag.String("region", "us-east-1", "aws region")
	access := flag.String("access", os.Getenv("AWS_ACCESS"), "aws access id")
	secret := flag.String("secret", os.Getenv("AWS_SECRET"), "aws secret key")

	flag.Parse()

	if len(flag.Args()) != 3 {
		flag.Usage()
		os.Exit(0)
	}

	args := flag.Args()

	builder := NewBuilder()
	builder.AwsRegion = *region
	builder.AwsAccess = *access
	builder.AwsSecret = *secret

	repo := args[0]
	name := fmt.Sprintf("%s-%s", args[1], args[2])
	builder.Build(repo, name)

	fmt.Printf("builder %+v\n", builder)
}
