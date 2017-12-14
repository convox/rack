package local

import (
	"fmt"
	"time"

	"github.com/convox/rack/structs"
	"github.com/pkg/errors"
)

const (
	RegistryCacheDuration = 1 * time.Hour
)

func (p *Provider) RegistryAdd(server, username, password string) (*structs.Registry, error) {
	log := p.logger("RegistryAdd").Append("server=%q username=%q", server, username)

	r := &structs.Registry{
		Server:   server,
		Username: username,
		Password: password,
	}

	key := fmt.Sprintf("registries/%s", server)

	if p.storageExists(key) {
		return nil, log.Error(fmt.Errorf("registry already exists: %s", server))
	}

	if err := p.storageStore(fmt.Sprintf("registries/%s", server), r); err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	return r, log.Success()
}

func (p *Provider) RegistryList() (structs.Registries, error) {
	log := p.logger("RegistryList")

	names, err := p.storageList("registries")
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	registries := make(structs.Registries, len(names))

	var r structs.Registry

	for i, name := range names {
		if err := p.storageLoad(fmt.Sprintf("registries/%s", name), &r, RegistryCacheDuration); err != nil {
			return nil, errors.WithStack(log.Error(err))
		}

		registries[i] = r
	}

	return registries, log.Success()
}

func (p *Provider) RegistryRemove(server string) error {
	log := p.logger("RegistryAdd").Append("server=%q", server)

	key := fmt.Sprintf("registries/%s", server)

	if !p.storageExists(key) {
		return log.Error(fmt.Errorf("no such registry: %s", server))
	}

	if err := p.storageDelete(key); err != nil {
		errors.WithStack(log.Error(err))
	}

	return log.Success()
}
