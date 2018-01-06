package aws

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/convox/rack/structs"
	docker "github.com/fsouza/go-dockerclient"
)

func (p *AWSProvider) RegistryAdd(server, username, password string) (*structs.Registry, error) {
	log := Logger.At("RegistryAdd").Namespace("server=%q username=%q", server, username).Start()

	if server == "" {
		return nil, fmt.Errorf("server must not be blank")
	}

	if username == "" {
		return nil, fmt.Errorf("username must not be blank")
	}

	if password == "" {
		return nil, fmt.Errorf("password must not be blank")
	}

	dc, err := docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		return nil, err
	}

	// validate login
	switch {
	case regexpECRHost.MatchString(server):
		system, err := p.describeStack(p.Rack)
		if err != nil {
			return nil, err
		}
		stackID := regexpStackID.FindStringSubmatch(*system.StackId)
		if len(stackID) < 3 {
			return nil, fmt.Errorf("invalid stack id %s", *system.StackId)
		}
		accountID := stackID[2]
		host := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", accountID, p.Region)

		if host == server {
			return nil, fmt.Errorf("can't add the rack's internal registry: %s", server)
		}

		if _, _, err := p.authECR(server, username, password); err != nil {
			return nil, fmt.Errorf("unable to authenticate")
		}
	default:
		_, err := dc.AuthCheck(&docker.AuthConfiguration{
			ServerAddress: server,
			Username:      username,
			Password:      password,
		})
		if err != nil {
			return nil, fmt.Errorf("unable to authenticate")
		}
	}

	r := structs.Registry{
		Server:   server,
		Username: username,
		Password: password,
	}

	data, err := json.Marshal(r)
	if err != nil {
		return nil, log.Error(err)
	}

	id := fmt.Sprintf("%x", sha256.New().Sum([]byte(server)))

	if err := p.SettingPut(fmt.Sprintf("system/registries/%s", id), string(data)); err != nil {
		return nil, log.Error(err)
	}

	registry := &structs.Registry{
		Server:   server,
		Username: username,
		Password: password,
	}

	return registry, log.Success()
}

func (p *AWSProvider) RegistryRemove(server string) error {
	log := Logger.At("RegistryRemove").Namespace("server=%q", server).Start()

	key := fmt.Sprintf("system/registries/%x", sha256.New().Sum([]byte(server)))

	if _, err := p.SettingExists(key); err != nil {
		return log.Error(fmt.Errorf("no such registry: %s", server))
	}

	if err := p.SettingDelete(key); err != nil {
		return log.Error(err)
	}

	return log.Success()
}

func (p *AWSProvider) RegistryList() (structs.Registries, error) {
	log := Logger.At("RegistryList").Start()

	objects, err := p.SettingList(structs.SettingListOptions{Prefix: "system/registries/"})
	if err != nil {
		return nil, log.Error(err)
	}

	registries := structs.Registries{}

	for _, o := range objects {
		data, err := p.SettingGet(o)
		if err != nil {
			return nil, log.Error(err)
		}

		var reg structs.Registry

		if err := json.Unmarshal([]byte(data), &reg); err != nil {
			return nil, log.Error(err)
		}

		registries = append(registries, reg)
	}

	sort.Sort(registries)

	return registries, log.Success()
}
