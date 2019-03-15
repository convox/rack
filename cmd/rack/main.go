package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/convox/rack/pkg/api"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}
}

func run() error {
	s, err := api.New()
	if err != nil {
		return err
	}

	s.Password = os.Getenv("PASSWORD")

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go handleSignals(s, ch)

	return s.Listen("https", ":5443")
}

func handleSignals(s *api.Server, ch <-chan os.Signal) {
	sig := <-ch

	fmt.Printf("ns=rack at=signal signal=%v terminate=true\n", sig)

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	s.Shutdown(ctx)

	os.Exit(0)
}
