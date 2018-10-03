package cli_test

import (
	"time"

	"github.com/convox/rack/pkg/structs"
)

func fxApp() *structs.App {
	return &structs.App{
		Name:       "app1",
		Generation: "2",
		Parameters: fxParameters(),
		Release:    "release1",
		Status:     "running",
	}
}

func fxAppGeneration1() *structs.App {
	return &structs.App{
		Name:       "app1",
		Generation: "1",
		Parameters: fxParameters(),
		Release:    "release1",
		Status:     "running",
	}
}

func fxAppUpdating() *structs.App {
	return &structs.App{
		Name:       "app1",
		Generation: "2",
		Parameters: fxParameters(),
		Release:    "release1",
		Status:     "updating",
	}
}

func fxBuild() *structs.Build {
	return &structs.Build{
		Id:          "build1",
		Description: "desc",
		Ended:       fxStarted.Add(2 * time.Minute),
		Manifest:    "manifest1\nmanifest2\n",
		Release:     "release1",
		Started:     fxStarted,
		Status:      "complete",
	}
}

func fxBuildCreated() *structs.Build {
	return &structs.Build{
		Id:     "build2",
		Status: "running",
	}
}

func fxBuildFailed() *structs.Build {
	return &structs.Build{
		Id:      "build3",
		Started: fxStarted,
		Status:  "failed",
	}
}

func fxBuildRunning() *structs.Build {
	return &structs.Build{
		Id:      "build4",
		Started: fxStarted,
		Status:  "running",
	}
}

func fxCertificate() *structs.Certificate {
	return &structs.Certificate{
		Id:         "cert1",
		Domain:     "example.org",
		Domains:    []string{"example.net", "example.com"},
		Expiration: time.Now().Add(49 * time.Hour).UTC(),
	}
}

func fxInstance() *structs.Instance {
	return &structs.Instance{
		Agent:     true,
		Cpu:       0.423,
		Id:        "instance1",
		Memory:    0.718,
		PrivateIp: "private",
		Processes: 3,
		PublicIp:  "public",
		Status:    "status",
		Started:   time.Now().UTC().Add(-48 * time.Hour),
	}
}

func fxLogs() []string {
	return []string{
		"log1",
		"log2",
	}
}

func fxLogsSystem() []string {
	return []string{
		"TIME system/aws/component log1",
		"TIME system/aws/component log2",
	}
}

func fxParameters() map[string]string {
	return map[string]string{
		"ParamFoo":      "value1",
		"ParamOther":    "value2",
		"ParamPassword": "****",
	}
}

func fxProcess() *structs.Process {
	return &structs.Process{
		Id:       "pid1",
		App:      "app1",
		Command:  "command",
		Cpu:      1.0,
		Host:     "host",
		Image:    "image",
		Instance: "instance",
		Memory:   2.0,
		Name:     "name",
		Ports:    []string{"1000", "2000"},
		Release:  "release1",
		Started:  time.Now().UTC().Add(-49 * time.Hour),
		Status:   "running",
	}
}

func fxProcessPending() *structs.Process {
	return &structs.Process{
		Id:       "pid1",
		App:      "app1",
		Command:  "command",
		Cpu:      1.0,
		Host:     "host",
		Image:    "image",
		Instance: "instance",
		Memory:   2.0,
		Name:     "name",
		Ports:    []string{"1000", "2000"},
		Release:  "release1",
		Started:  time.Now().UTC().Add(-49 * time.Hour),
		Status:   "pending",
	}
}

func fxRegistry() *structs.Registry {
	return &structs.Registry{
		Server:   "registry1",
		Username: "username",
		Password: "password",
	}
}

func fxRelease() *structs.Release {
	return &structs.Release{
		Id:          "release1",
		App:         "app1",
		Build:       "build1",
		Env:         "FOO=bar\nBAZ=quux",
		Manifest:    "services:\n  web:\n    build: .",
		Created:     time.Now().UTC().Add(-49 * time.Hour),
		Description: "description1",
	}
}

func fxRelease2() *structs.Release {
	return &structs.Release{
		Id:          "release2",
		App:         "app1",
		Build:       "build1",
		Env:         "FOO=bar\nBAZ=quux",
		Manifest:    "manifest",
		Created:     time.Now().UTC().Add(-49 * time.Hour),
		Description: "description2",
	}
}

func fxRelease3() *structs.Release {
	return &structs.Release{
		Id:       "release3",
		App:      "app1",
		Build:    "build1",
		Env:      "FOO=bar\nBAZ=quux",
		Manifest: "manifest",
		Created:  time.Now().UTC().Add(-49 * time.Hour),
	}
}

func fxResource() *structs.Resource {
	return &structs.Resource{
		Name:       "resource1",
		Parameters: map[string]string{"k1": "v1", "k2": "v2", "Url": "https://other.example.org/path"},
		Status:     "status",
		Type:       "type",
		Url:        "https://example.org/path",
		Apps:       structs.Apps{*fxApp(), *fxApp()},
	}
}

func fxResourceType() structs.ResourceType {
	return structs.ResourceType{
		Name: "type1",
		Parameters: structs.ResourceParameters{
			{Default: "def1", Description: "desc1", Name: "Param1"},
			{Default: "def2", Description: "desc2", Name: "Param2"},
		},
	}
}

func fxService() *structs.Service {
	return &structs.Service{
		Name:   "service1",
		Count:  1,
		Cpu:    2,
		Domain: "domain",
		Memory: 3,
		Ports: []structs.ServicePort{
			{Balancer: 1, Certificate: "cert1", Container: 2},
			{Balancer: 1, Certificate: "cert1", Container: 2},
		},
	}
}

func fxSystem() *structs.System {
	return &structs.System{
		Count:      1,
		Domain:     "domain",
		Name:       "name",
		Outputs:    map[string]string{"k1": "v1", "k2": "v2"},
		Parameters: map[string]string{"Autoscale": "Yes", "ParamFoo": "value1", "ParamOther": "value2"},
		Provider:   "provider",
		Region:     "region",
		Status:     "running",
		Type:       "type",
		Version:    "20180901000000",
	}
}

func fxSystemClassic() *structs.System {
	return &structs.System{
		Count:      1,
		Domain:     "domain",
		Name:       "name",
		Outputs:    map[string]string{"k1": "v1", "k2": "v2"},
		Parameters: map[string]string{"ParamFoo": "value1", "ParamOther": "value2"},
		Provider:   "provider",
		Region:     "region",
		Status:     "running",
		Type:       "type",
		Version:    "20180101000000",
	}
}

func fxSystemLocal() *structs.System {
	return &structs.System{
		Name:     "convox",
		Provider: "local",
		Status:   "running",
		Version:  "dev",
	}
}
