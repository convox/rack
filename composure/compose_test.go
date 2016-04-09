package composure

import (
	"testing"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/docker/libcompose/config"
	"github.com/docker/libcompose/docker"
	"github.com/docker/libcompose/project"
	"github.com/docker/libcompose/yaml"
)

func TestNewProject(t *testing.T) {
	project, err := docker.NewProject(&docker.Context{
		Context: project.Context{
			ComposeFiles: []string{"fixtures/httpd.yml"},
			ProjectName:  "httpd",
		},
	})
	assert.Nil(t, err)

	s, ok := project.Configs.Get("web")
	assert.True(t, ok)
	assert.EqualValues(t, &config.ServiceConfig{
		Build:         "",
		CapAdd:        []string(nil),
		CapDrop:       []string(nil),
		CgroupParent:  "",
		CPUQuota:      0,
		CPUSet:        "",
		CPUShares:     0,
		Command:       yaml.Command{},
		ContainerName: "",
		Devices:       []string(nil),
		DNS:           yaml.Stringorslice{},
		DNSSearch:     yaml.Stringorslice{},
		Dockerfile:    "",
		DomainName:    "",
		Entrypoint:    yaml.Command{},
		EnvFile:       yaml.Stringorslice{},
		Environment:   yaml.MaporEqualSlice{},
		Hostname:      "",
		Image:         "httpd",
		Labels:        yaml.SliceorMap{},
		Links:         yaml.MaporColonSlice{},
		LogDriver:     "",
		MacAddress:    "",
		MemLimit:      0,
		MemSwapLimit:  0,
		Name:          "",
		Net:           "",
		Pid:           "",
		Uts:           "",
		Ipc:           "",
		Ports:         []string{"80:80"},
		Privileged:    false,
		Restart:       "",
		ReadOnly:      false,
		StdinOpen:     false,
		SecurityOpt:   []string(nil),
		Tty:           false,
		User:          "",
		VolumeDriver:  "",
		Volumes:       []string(nil),
		VolumesFrom:   []string(nil),
		WorkingDir:    "",
		Expose:        []string(nil),
		ExternalLinks: []string(nil),
		LogOpt:        map[string]string(nil),
		ExtraHosts:    []string(nil),
		Ulimits:       yaml.Ulimits{Elements: []yaml.Ulimit(nil)},
	}, s)
}
