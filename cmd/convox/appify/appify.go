package appify

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/convox/rack/cmd/convox/appify/templates"
	"github.com/convox/rack/cmd/convox/helpers"
)

// Framework interface for different languages and frameworks
type Framework interface {
	Appify() error
	Setup(string) error
}

func writeFile(path string, data []byte, mode os.FileMode) error {
	fmt.Printf("Writing %s... ", path)

	if helpers.Exists(path) {
		fmt.Println("EXISTS")
		return nil
	}

	// make the containing directory
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	if err := ioutil.WriteFile(path, data, mode); err != nil {
		return err
	}

	fmt.Println("OK")

	return nil
}

func writeAsset(path, templateName string, input map[string]interface{}) error {
	data, err := templates.Asset(templateName)
	if err != nil {
		return err
	}

	info, err := templates.AssetInfo(templateName)
	if err != nil {
		return err
	}

	if input != nil {
		tmpl, err := template.New(templateName).Parse(string(data))
		if err != nil {
			return err
		}

		var formation bytes.Buffer

		err = tmpl.Execute(&formation, input)
		if err != nil {
			return err
		}

		data = formation.Bytes()
	}

	return writeFile(path, data, info.Mode())
}
