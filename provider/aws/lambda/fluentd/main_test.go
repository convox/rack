package main

import (
	"fmt"
	"testing"
)

func TestDecodeLogLine(t *testing.T) {
	_, err := decodeLogLine(`app:RPHXAIGWFGX/9c4d087026c3 {"key":"value", "key2": "asdf1"}`)
	if err != nil {
		t.Error("Failed to parse log line:", err)
	}
}

func TestDecodeWithoutSpaces(t *testing.T) {
	_, err := decodeLogLine(`app:RFPWOWFYQMP/e9325df7e748 {"@timestamp":"2016-08-12T15:29:19Z","@version":1,"@event_name":"worker:success"}`)
	if err != nil {
		t.Error("Failed to parse log line:", err)
	}
}

func TestDecodeWithSpaces(t *testing.T) {
	_, err := decodeLogLine(`app:RFPWOWFYQMP/e9325df7e748 {"@timestamp": "2016-08-12T15:29:19Z", "@version": 1, "@event_name": "worker:success"}`)
	if err != nil {
		t.Error("Failed to parse log line:", err)
	}
}

func TestDecodeInvalidJSON(t *testing.T) {
	_, err := decodeLogLine(`app:RFPWOWFYQMP/e9325df7e748 ta":{"event":{"name":"listings","id":"1151672","timestamp":"2016-08-12T15:29:19Z"}}}`)
	if err != nil {
		fmt.Println("Got error as expected:", err)
	}
}
