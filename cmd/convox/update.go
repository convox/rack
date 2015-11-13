package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/equinox-io/equinox"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "update",
		Description: "update the cli",
		Usage:       "",
		Action:      cmdUpdate,
	})
}

var publicKey = []byte(`
-----BEGIN ECDSA PUBLIC KEY-----
MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEFtnvjr4tRncwDB+SiCVORcprHlcBLPy9
tYsnnPefR196tYvG62trcNZXF3qw2UCDkc9eWMnloiTVsKfEuUy3TTU3KxwOzD38
77z5PI4u680jtKAm0zUIefrsnwrYWqUW
-----END ECDSA PUBLIC KEY-----
`)

func cmdUpdate(c *cli.Context) {
	client, err := updateClient()
	if err != nil {
		stdcli.Error(err)
	}

	stdcli.Spinner.Prefix = "Updating: "
	stdcli.Spinner.Start()

	opts := equinox.Options{
		CurrentVersion: Version,
		Channel:        "stable",
		HTTPClient:     client,
	}
	if err := opts.SetPublicKeyPEM(publicKey); err != nil {
		stdcli.Error(err)
		return
	}

	// check for update
	r, err := equinox.Check("app_i8m2L26DxKL", opts)
	switch {
	case err == equinox.NotAvailableErr:
		fmt.Println("\x08\x08Already up to date")
		return
	case err != nil:
		stdcli.Error(err)
		return
	}

	// apply update
	err = r.Apply()
	if err != nil {
		stdcli.Error(err)
		return
	}

	stdcli.Spinner.Stop()
	fmt.Printf("\x08\x08OK, %s\n", r.ReleaseVersion)
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
