package local

import (
	"fmt"

	"github.com/convox/rack/structs"
)

func (p *Provider) SettingDelete(name string) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) SettingGet(name string) (string, error) {
	return "", fmt.Errorf("unimplemented")
}

func (p *Provider) SettingList(opts structs.SettingListOptions) ([]string, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SettingPut(name, value string) error {
	return fmt.Errorf("unimplemented")
}
