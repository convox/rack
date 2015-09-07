package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Error struct {
	Error string `json:"error"`
}

func responseError(res *http.Response) error {
	if res.StatusCode < 400 {
		return nil
	}

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return fmt.Errorf("error reading response body: %s", err)
	}

	var e Error

	err = json.Unmarshal(data, &e)

	if err != nil {
		return fmt.Errorf("response status: %d", res.StatusCode)
	}

	return fmt.Errorf(e.Error)
}
