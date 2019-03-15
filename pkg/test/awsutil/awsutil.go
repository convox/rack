package awsutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

// Request represents an expected AWS API Operation.
type Request struct {
	Method     string
	RequestURI string
	Operation  string
	Body       string
}

func (r *Request) String() string {
	body := formatBody(strings.NewReader(r.Body))
	return fmt.Sprintf("Method: %q,\nRequestURI: %q,\nOperation: %q,\nBody: `%s`,", r.Method, r.RequestURI, r.Operation, body)
}

// Response represents a predefined response.
type Response struct {
	StatusCode int
	Body       string
}

// Cycle represents a request-response cycle.
type Cycle struct {
	Request  Request
	Response Response
}

// Handler is an http.Handler that will play back cycles.
type Handler struct {
	cycles []Cycle
	count  int
}

// NewHandler returns a new Handler instance.
func NewHandler(c []Cycle) *Handler {
	return &Handler{cycles: c}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	match := Request{
		Method:     r.Method,
		RequestURI: r.URL.RequestURI(),
		Operation:  r.Header.Get("X-Amz-Target"),
		Body:       string(b),
	}

	if len(h.cycles) == 0 {
		fmt.Println("No cycles remaining to replay.")
		fmt.Println(match.String())
		w.WriteHeader(404)
		return
	}

	cycle := h.cycles[0]

	if cycle.Request.Method == "" {
		cycle.Request.Method = "POST"
	}

	var matched bool

	matched = (cycle.Request.Body == "ignore")

	// treat "/a string with slashes/" as a regex
	if !matched && strings.HasPrefix(cycle.Request.Body, "/") {
		size := len(cycle.Request.Body)
		trimmed := cycle.Request.Body[1 : size-1]
		matched, _ = regexp.MatchString(trimmed, match.Body)
	}

	if !matched {
		matched = (cycle.Request.String() == match.String())
	}

	if matched {
		w.WriteHeader(cycle.Response.StatusCode)
		io.WriteString(w, cycle.Response.Body)
	} else {
		fmt.Printf("Request %d does not match next cycle.\n", h.count)
		fmt.Println("CYCLE REQUEST:")
		fmt.Println(cycle.Request.String())
		fmt.Println("ACTUAL REQUEST:")
		fmt.Println(match.String())
		w.WriteHeader(404)
	}

	h.cycles = h.cycles[1:]
	h.count++
}

func formatBody(r io.Reader) string {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}

	s, err := formatJSON(bytes.NewReader(b))
	if err == nil {
		return s
	}

	return string(b)
}

func formatJSON(r io.Reader) (string, error) {
	var body map[string]interface{}
	if err := json.NewDecoder(r).Decode(&body); err != nil {
		return "", err
	}

	raw, err := json.MarshalIndent(&body, "", "  ")
	if err != nil {
		return "", err
	}

	return string(raw), nil
}
