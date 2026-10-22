package aws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/convox/rack/pkg/helpers"
)

var (
	certificateRegistrarClient = &http.Client{Timeout: 10 * time.Second}
	certificateRegistrarURL    = "https://certbot.convox.com/validation"
)

type registrarIdentity struct {
	URL     string      `json:"url"`
	Headers http.Header `json:"headers"`
	Body    string      `json:"body"`
}

type registrarRequest struct {
	Identity registrarIdentity `json:"identity"`
	Domain   string            `json:"domain"`
	Name     string            `json:"name"`
	Value    string            `json:"value"`
}

type registrarError struct {
	status  int
	message string
}

func (e registrarError) Error() string {
	return fmt.Sprintf("registrar %d: %s", e.status, e.message)
}

func (e registrarError) retry() bool {
	if e.status >= 500 {
		return true
	}

	return e.status == 400 && (e.message == "load balancer not found" || e.message == "load balancer not registered")
}

func (e registrarError) support() bool {
	return e.status == 409 || e.status == 429
}

// registrarIdentity signs a GetCallerIdentity request the registrar forwards to STS to learn the
// caller's account. X-Convox-Registrar is signed, so the request is only good for the registrar.
func (p *Provider) registrarIdentity() (*registrarIdentity, error) {
	s, err := helpers.NewSession()
	if err != nil {
		return nil, err
	}

	req, _ := sts.New(s, p.config()).GetCallerIdentityRequest(nil)
	req.HTTPRequest.Header.Set("X-Convox-Registrar", "convox.site")

	if err := req.Sign(); err != nil {
		return nil, err
	}

	body := req.GetBody()

	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	return &registrarIdentity{
		URL:     req.HTTPRequest.URL.String(),
		Headers: req.HTTPRequest.Header,
		Body:    string(data),
	}, nil
}

func (p *Provider) registrarPublish(ctx context.Context, domain, name, value string) error {
	id, err := p.registrarIdentity()
	if err != nil {
		return err
	}

	data, err := json.Marshal(registrarRequest{Identity: *id, Domain: domain, Name: name, Value: value})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, certificateRegistrarURL, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := certificateRegistrarClient.Do(req)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, 64*1024))
	if cerr := res.Body.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}

	if res.StatusCode == http.StatusOK {
		return nil
	}

	var e struct {
		Error string `json:"error"`
	}

	if err := json.Unmarshal(body, &e); err != nil || e.Error == "" {
		e.Error = strings.TrimSpace(string(body))
	}

	if e.Error == "" {
		e.Error = http.StatusText(res.StatusCode)
	}

	return registrarError{status: res.StatusCode, message: e.Error}
}
