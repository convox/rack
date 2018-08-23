package k8s

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/convox/rack/pkg/structs"
	ac "k8s.io/api/core/v1"
	ae "k8s.io/apimachinery/pkg/api/errors"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *Provider) RegistryAdd(server, username, password string) (*structs.Registry, error) {
	dc, err := p.dockerConfigLoad("registries")
	if err != nil {
		return nil, err
	}

	if dc.Auths == nil {
		dc.Auths = map[string]dockerConfigAuth{}
	}

	dc.Auths[server] = dockerConfigAuth{
		Auth: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password))),
	}

	if err := p.dockerConfigSave("registries", dc); err != nil {
		return nil, err
	}

	r := &structs.Registry{
		Server:   server,
		Username: username,
		Password: password,
	}

	return r, nil
}

func (p *Provider) RegistryList() (structs.Registries, error) {
	dc, err := p.dockerConfigLoad("registries")
	if err != nil {
		return nil, err
	}

	rs := structs.Registries{}

	if dc.Auths != nil {
		for host, auth := range dc.Auths {
			data, err := base64.StdEncoding.DecodeString(auth.Auth)
			if err != nil {
				return nil, err
			}

			parts := strings.SplitN(string(data), ":", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid auth for registry: %s", host)
			}

			rs = append(rs, structs.Registry{
				Server:   host,
				Username: parts[0],
				Password: parts[1],
			})
		}
	}

	return rs, nil
}

func (p *Provider) RegistryRemove(server string) error {
	dc, err := p.dockerConfigLoad("registries")
	if err != nil {
		return err
	}
	if dc.Auths == nil {
		return fmt.Errorf("no such registry: %s", server)
	}
	if _, ok := dc.Auths[server]; !ok {
		return fmt.Errorf("no such registry: %s", server)
	}

	delete(dc.Auths, server)

	if err := p.dockerConfigSave("registries", dc); err != nil {
		return err
	}

	return nil
}

type dockerConfig struct {
	Auths map[string]dockerConfigAuth `json:"auths"`
}

type dockerConfigAuth struct {
	Auth string `json:"auth"`
}

func (p *Provider) dockerConfigLoad(secret string) (*dockerConfig, error) {
	s, err := p.Cluster.CoreV1().Secrets(p.Rack).Get(secret, am.GetOptions{})
	if ae.IsNotFound(err) {
		return &dockerConfig{}, nil
	}
	if err != nil {
		return nil, err
	}
	if s.Type != ac.SecretTypeDockerConfigJson {
		return nil, fmt.Errorf("invalid type for secret: %s", secret)
	}
	data, ok := s.Data[".dockerconfigjson"]
	if !ok {
		return nil, fmt.Errorf("invalid data for secret: %s", secret)
	}

	var dc dockerConfig

	if err := json.Unmarshal(data, &dc); err != nil {
		return nil, err
	}

	return &dc, nil
}

func (p *Provider) dockerConfigSave(secret string, dc *dockerConfig) error {
	data, err := json.Marshal(dc)
	if err != nil {
		return err
	}

	sd := map[string][]byte{
		".dockerconfigjson": data,
	}

	s, err := p.Cluster.CoreV1().Secrets(p.Rack).Get(secret, am.GetOptions{})
	if ae.IsNotFound(err) {
		_, err := p.Cluster.CoreV1().Secrets(p.Rack).Create(&ac.Secret{
			ObjectMeta: am.ObjectMeta{
				Name: "registries",
				Labels: map[string]string{
					"system": "convox",
					"rack":   p.Rack,
				},
			},
			Type: ac.SecretTypeDockerConfigJson,
			Data: sd,
		})
		return err
	}

	s.Data = sd

	_, err = p.Cluster.CoreV1().Secrets(p.Rack).Update(s)
	if err != nil {
		return err
	}

	return nil
}
