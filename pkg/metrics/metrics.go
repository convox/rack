package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Metrics struct {
	url string
}

func New(url string) *Metrics {
	return &Metrics{url: url}
}

func (m *Metrics) Post(name string, attrs map[string]interface{}) error {
	data, err := json.Marshal(attrs)
	if err != nil {
		return err
	}

	res, err := http.Post(fmt.Sprintf("%s/%s", m.url, name), "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}
