package cli

import (
	"fmt"

	"github.com/convox/rack/pkg/crypt"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("encrypt", "encrypt data using key", Encrypt, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			stdcli.StringFlag("key", "", "key"),
			stdcli.StringFlag("data", "", "data"),
		},
		Usage: "",
	})

	register("decrypt", "decrypt data using key", Decrypt, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			stdcli.StringFlag("key", "", "key"),
			stdcli.StringFlag("data", "", "data"),
		},
		Usage: "",
	})
}

func Encrypt(_ sdk.Interface, c *stdcli.Context) error {

	key := c.String("key")
	data := c.String("data")

	if key == "" || data == "" {
		return fmt.Errorf("key and data must be non empty")
	}

	val, err := crypt.Encrypt(key, []byte(data))
	if err != nil {
		return err
	}

	fmt.Println(val)
	return nil
}

func Decrypt(_ sdk.Interface, c *stdcli.Context) error {
	key := c.String("key")
	data := c.String("data")

	if key == "" || data == "" {
		return fmt.Errorf("key and data must be non empty")
	}

	val, err := crypt.Decrypt(key, data)
	if err != nil {
		return err
	}

	fmt.Println(string(val))
	return nil
}
