package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os/exec"

	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/equinox-io/equinox"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "update",
		Description: "update the cli",
		Usage:       "",
		Action:      cmdUpdate,
		Flags:       []cli.Flag{rackFlag},
	})
}

var publicKey = []byte(`
-----BEGIN ECDSA PUBLIC KEY-----
MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEFtnvjr4tRncwDB+SiCVORcprHlcBLPy9
tYsnnPefR196tYvG62trcNZXF3qw2UCDkc9eWMnloiTVsKfEuUy3TTU3KxwOzD38
77z5PI4u680jtKAm0zUIefrsnwrYWqUW
-----END ECDSA PUBLIC KEY-----
`)

func cmdUpdate(c *cli.Context) error {
	client, err := updateClient()
	if err != nil {
		return stdcli.Error(err)
	}

	stdcli.Spinner.Prefix = "Updating convox/proxy: "
	stdcli.Spinner.Start()

	if err := updateProxy(); err != nil {
		fmt.Printf("\x08\x08FAILED\n")
	} else {
		fmt.Printf("\x08\x08OK\n")
	}
	stdcli.Spinner.Stop()

	stdcli.Spinner.Prefix = "Updating convox: "
	stdcli.Spinner.Start()

	opts := equinox.Options{
		CurrentVersion: Version,
		Channel:        "stable",
		HTTPClient:     client,
	}
	if err := opts.SetPublicKeyPEM(publicKey); err != nil {
		return stdcli.Error(err)
	}

	// check for update
	r, err := equinox.Check("app_i8m2L26DxKL", opts)
	switch {
	case err == equinox.NotAvailableErr:
		fmt.Println("\x08\x08Already up to date")
		return nil
	case err != nil:
		return stdcli.Error(err)
	}

	// apply update
	err = r.Apply()
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Printf("\x08\x08OK, %s\n", r.ReleaseVersion)
	stdcli.Spinner.Stop()

	return nil
}

func updateClient() (*http.Client, error) {
	root, err := Asset("data/root.pem")
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(root)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool},
		},
	}

	return client, nil
}

func updateProxy() error {
	cmd := exec.Command("docker", "pull", "convox/proxy")
	return cmd.Run()
}
