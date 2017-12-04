package aws

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"

	"github.com/convox/rack/crypt"
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

	enc, err := p.KeyEncrypt(data)
	if err != nil {
		return nil, log.Error(err)
	}

	if _, err := p.ObjectStore(fmt.Sprintf("system/registries/%s", id), bytes.NewReader(enc), structs.ObjectOptions{Public: false}); err != nil {
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

		data, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, log.Error(err)
		}

		dec, err := p.KeyDecrypt(data)
		if err != nil {
			if err.Error() != "invalid ciphertext" {
				return nil, err
			}

			dec = data
		}

		var reg structs.Registry

		err = json.Unmarshal(dec, &reg)
		if err != nil {
			return nil, log.Error(err)
		}

		// if the registry is unencrypted, encrypt it
		if bytes.Compare(data, dec) == 0 {
			if _, err := p.RegistryAdd(reg.Server, reg.Username, reg.Password); err != nil {
				return nil, err
			}
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
			if d, err := crypt.New().Decrypt(p.EncryptionKey, data); err == nil {
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
