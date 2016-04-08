package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "ssl",
		Action:      cmdSSLList,
		Description: "manage SSL certificates",
		Flags: []cli.Flag{
			appFlag,
		},
		Subcommands: []cli.Command{
			{
				Name:        "create",
				Description: "create a new SSL listener",
				Usage:       "<process:port> [<foo.pub> <foo.key>|<arn>]",
				Action:      cmdSSLCreate,
				Flags: []cli.Flag{
					appFlag,
					cli.StringFlag{
						Name:  "chain",
						Usage: "Intermediate certificate chain.",
					},
					cli.BoolFlag{
						Name:  "secure",
						Usage: "Use a TLS-encrypted listener.",
					},
					cli.BoolFlag{
						Name:  "self-signed",
						Usage: "Generate a self-signed cert.",
					},
				},
			},
			{
				Name:        "delete",
				Description: "delete an SSL listener",
				Usage:       "<process:port>",
				Action:      cmdSSLDelete,
				Flags: []cli.Flag{
					appFlag,
				},
			},
			{
				Name:        "update",
				Description: "upload a replacement ssl certificate",
				Usage:       "<process:port> [<foo.pub> <foo.key>|<arn>]",
				Action:      cmdSSLUpdate,
				Flags: []cli.Flag{
					appFlag,
					cli.StringFlag{
						Name:  "chain",
						Usage: "Intermediate certificate chain.",
					},
				},
			},
		},
	})
}

func cmdSSLCreate(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) < 1 {
		stdcli.Usage(c, "create")
		return
	}

	target := c.Args()[0]

	parts := strings.Split(target, ":")

	if len(parts) != 2 {
		stdcli.Error(fmt.Errorf("target must be process:port"))
		return
	}

	var pub []byte
	var key []byte
	var arn string

	switch len(c.Args()) {
	case 1:
		if c.Bool("self-signed") {
			formation, err := rackClient(c).ListFormation(app)

			if err != nil {
				stdcli.Error(err)
				return
			}

			host := ""

			for _, entry := range formation {
				if entry.Name == parts[0] {
					host = entry.Balancer
				}
			}

			if host == "" {
				stdcli.Error(fmt.Errorf("no balancer for process: %s", parts[0]))
				return
			}

			pub, key, err = generateSelfSignedCertificate(app, host)

			if err != nil {
				stdcli.Error(err)
				return
			}
		} else {
			stdcli.Usage(c, "create")
			return
		}
	case 2:
		arn = c.Args()[1]
	case 3:
		pub, err = ioutil.ReadFile(c.Args()[1])

		if err != nil {
			stdcli.Error(err)
			return
		}

		key, err = ioutil.ReadFile(c.Args()[2])

		if err != nil {
			stdcli.Error(err)
			return
		}
	default:
		stdcli.Usage(c, "create")
		return
	}

	chain := ""

	if chainFile := c.String("chain"); chainFile != "" {
		data, err := ioutil.ReadFile(chainFile)

		if err != nil {
			stdcli.Error(err)
			return
		}

		chain = string(data)
	}

	fmt.Printf("Creating SSL listener %s... ", target)

	_, err = rackClient(c).CreateSSL(app, parts[0], parts[1], arn, string(pub), string(key), chain, c.Bool("secure"))

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("Done.")
}

func cmdSSLDelete(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) != 1 {
		stdcli.Usage(c, "delete")
		return
	}

	target := c.Args()[0]

	parts := strings.Split(target, ":")

	if len(parts) != 2 {
		stdcli.Error(fmt.Errorf("target must be process:port"))
		return
	}

	fmt.Printf("Deleting SSL listener %s... ", target)

	_, err = rackClient(c).DeleteSSL(app, parts[0], parts[1])

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("Done.")
}

func cmdSSLList(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	ssls, err := rackClient(c).ListSSL(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	t := stdcli.NewTable("TARGET", "EXPIRES", "DOMAIN")

	for _, ssl := range *ssls {
		t.AddRow(fmt.Sprintf("%s:%d", ssl.Process, ssl.Port), humanizeTime(ssl.Expiration), ssl.Domain)
	}

	t.Print()
}

func cmdSSLUpdate(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) < 1 {
		stdcli.Usage(c, "create")
		return
	}

	target := c.Args()[0]

	parts := strings.Split(target, ":")

	if len(parts) != 2 {
		stdcli.Error(fmt.Errorf("target must be process:port"))
		return
	}

	var pub []byte
	var key []byte
	var arn string

	switch len(c.Args()) {
	case 2:
		arn = c.Args()[1]
	case 3:
		pub, err = ioutil.ReadFile(c.Args()[1])

		if err != nil {
			stdcli.Error(err)
			return
		}

		key, err = ioutil.ReadFile(c.Args()[2])

		if err != nil {
			stdcli.Error(err)
			return
		}
	default:
		stdcli.Usage(c, "update")
		return
	}

	chain := ""

	if chainFile := c.String("chain"); chainFile != "" {
		data, err := ioutil.ReadFile(chainFile)

		if err != nil {
			stdcli.Error(err)
			return
		}

		chain = string(data)
	}

	fmt.Printf("Updating SSL listener %s... ", target)

	_, err = rackClient(c).UpdateSSL(app, parts[0], parts[1], arn, string(pub), string(key), chain)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("Done.")
}

func generateSelfSignedCertificate(app, host string) ([]byte, []byte, error) {
	rkey, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		return nil, nil, err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   host,
			Organization: []string{app},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{host},
	}

	data, err := x509.CreateCertificate(rand.Reader, &template, &template, &rkey.PublicKey, rkey)

	if err != nil {
		return nil, nil, err
	}

	pub := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: data})
	key := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rkey)})

	return pub, key, nil
}
