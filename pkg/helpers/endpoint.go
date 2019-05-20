package helpers

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

func EndpointCheck(url string) error {
	ht := *(http.DefaultTransport.(*http.Transport))
	ht.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	hc := &http.Client{Timeout: 2 * time.Second, Transport: &ht}

	res, err := hc.Get(fmt.Sprintf("%s/apps", url))
	if err == nil && res.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("check failed")
}

func EndpointWait(url string) error {
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-tick.C:
			if err := EndpointCheck(url); err == nil {
				return nil
			}
		case <-timeout:
			return fmt.Errorf("timeout")
		}
	}
}
