package sdk

import (
	"net/url"
	"os"
)

const (
	sortableTime = "20060102.150405.000000000"
)

func New(endpoint string) (*Client, error) {
	u, err := url.Parse(coalesce(endpoint, "https://rack.convox"))
	if err != nil {
		return nil, err
	}

	return &Client{Debug: os.Getenv("CONVOX_DEBUG") == "true", Endpoint: u, Version: "dev"}, nil
}

func NewFromEnv() (*Client, error) {
	return New(os.Getenv("RACK_URL"))
}
