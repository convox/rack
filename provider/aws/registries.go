package aws

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"

	"github.com/convox/rack/api/crypt"
	"github.com/convox/rack/api/structs"
	docker "github.com/fsouza/go-dockerclient"
)

func (p *AWSProvider) RegistryAdd(server, username, password string) (*structs.Registry, error) {
	log := Logger.At("RegistryAdd").Namespace("server=%q username=%q", server, username).Start()

	dc, err := docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		return nil, err
	}

	_, err = dc.AuthCheck(&docker.AuthConfiguration{
		ServerAddress: server,
		Username:      username,
		Password:      password,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to authenticate")
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

	_, err = p.ObjectStore(fmt.Sprintf("system/registries/%s", id), bytes.NewReader(data), structs.ObjectOptions{Public: false})
	if err != nil {
		return nil, log.Error(err)
	}

	registry := &structs.Registry{
		Server:   server,
		Username: username,
		Password: password,
	}

	log.Success()
	return registry, nil
}

func (p *AWSProvider) RegistryDelete(server string) error {
	log := Logger.At("RegistryDelete").Namespace("server=%q", server).Start()

	key := fmt.Sprintf("system/registries/%x", sha256.New().Sum([]byte(server)))

	if !p.ObjectExists(key) {
		return log.Error(fmt.Errorf("no such registry: %s", server))
	}

	if err := p.ObjectDelete(key); err != nil {
		return log.Error(err)
	}

	log.Success()
	return nil
}

func (p *AWSProvider) RegistryList() (structs.Registries, error) {
	log := Logger.At("RegistryDelete").Start()

	if err := p.migrateClassicAuth(); err != nil {
		return nil, log.Error(err)
	}

	objects, err := p.ObjectList("system/registries/")
	if err != nil {
		return nil, log.Error(err)
	}

	registries := structs.Registries{}

	for _, o := range objects {
		r, err := p.ObjectFetch(o)
		if err != nil {
			return nil, log.Error(err)
		}

		defer r.Close()

		var reg structs.Registry

		err = json.NewDecoder(r).Decode(&reg)
		if err != nil {
			return nil, log.Error(err)
		}

		registries = append(registries, reg)
	}

	sort.Sort(registries)

	log.Success()
	return registries, nil
}

func (p *AWSProvider) migrateClassicAuth() error {
	r, err := p.ObjectFetch("env")
	if err != nil && !ErrorNotFound(err) {
		return err
	}

	data := []byte("{}")

	if r != nil {
		defer r.Close()

		d, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}
		data = d

		if p.EncryptionKey != "" {
			cr := crypt.New(p.Region, p.Access, p.Secret)

			if d, err := cr.Decrypt(p.EncryptionKey, data); err == nil {
				data = d
			}
		}
	}

	var env map[string]string
	err = json.Unmarshal(data, &env)
	if err != nil {
		return err
	}

	type authEntry struct {
		Username string
		Password string
	}

	auth := map[string]authEntry{}

	if ea, ok := env["DOCKER_AUTH_DATA"]; ok {
		if err := json.Unmarshal([]byte(ea), &auth); err != nil {
			return err
		}
	}

	if len(auth) == 0 {
		return nil
	}

	for server, entry := range auth {
		if _, err := p.RegistryAdd(server, entry.Username, entry.Password); err != nil {
			return err
		}
	}

	delete(env, "DOCKER_AUTH_DATA")

	data, err = json.Marshal(env)
	if err != nil {
		return err
	}

	_, err = p.ObjectStore("env", bytes.NewReader(data), structs.ObjectOptions{Public: false})
	if err != nil {
		return err
	}

	return nil
}
