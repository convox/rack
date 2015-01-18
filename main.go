package main

import (
	"flag"
	"fmt"
)

func main() {
	cluster := flag.String("cluster", "", "cluster name")
	app := flag.String("app", "", "app name")
	flag.Parse()

	fmt.Printf("cluster %+v\n", cluster)
	fmt.Printf("app %+v\n", app)
}
