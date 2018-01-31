package router

import (
	"fmt"
	"net/http"
	"os"
)

type logTransport struct {
	http.RoundTripper
}

func (t logTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	fmt.Printf("ns=convox.router at=proxy type=http target=%q\n", req.URL)

	return t.RoundTripper.RoundTrip(req)
}

func logError(err error) {
	fmt.Printf("err = %+v\n", err)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
	}
}
