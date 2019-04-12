package k8s

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path"
	"sort"
	"strings"

	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/structs"
	cc "github.com/convox/rack/provider/k8s/pkg/client/clientset/versioned/typed/convox/v1"
)

const (
	ScannerStartSize = 4096
	ScannerMaxSize   = 1024 * 1024
)

func (p *Provider) convoxClient() (cc.ConvoxV1Interface, error) {
	return cc.NewForConfig(p.Config)
}

func (p *Provider) serviceEnvironment(app, release string, s manifest.Service) (map[string]string, []string, error) {
	static := map[string]string{}
	secret := []string{}

	r, err := p.ReleaseGet(app, release)
	if err != nil {
		return nil, nil, err
	}

	env := structs.Environment{}

	if err := env.Load([]byte(r.Env)); err != nil {
		return nil, nil, err
	}

	for _, e := range s.Environment {
		parts := strings.SplitN(e, "=", 2)

		switch len(parts) {
		case 1:
			secret = append(secret, parts[0])
		case 2:
			if _, ok := env[parts[0]]; ok {
				secret = append(secret, parts[0])
			} else {
				static[parts[0]] = parts[1]
			}
		default:
			return nil, nil, fmt.Errorf("invalid environment: %s\n", e)
		}
	}

	return static, secret, nil
}

func (p *Provider) systemEnvironment(app, release string) (map[string]string, error) {
	senv := map[string]string{
		"APP":      app,
		"RACK":     p.Rack,
		"RACK_URL": fmt.Sprintf("https://convox:%s@api.%s.svc.cluster.local:5443", p.Password, p.Rack),
		"RELEASE":  release,
	}

	r, err := p.ReleaseGet(app, release)
	if err != nil {
		return nil, err
	}

	if r.Build != "" {
		b, err := p.BuildGet(app, r.Build)
		if err != nil {
			return nil, err
		}

		senv["BUILD"] = b.Id
		senv["BUILD_DESCRIPTION"] = b.Description
	}

	return senv, nil
}

func dockerSystemId() (string, error) {
	data, err := exec.Command("docker", "system", "info").CombinedOutput()
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "ID: ") {
			return strings.ToLower(strings.TrimPrefix(line, "ID: ")), nil
		}
	}

	return "", fmt.Errorf("could not find docker system id")
}

func envName(s string) string {
	return strings.Replace(strings.ToUpper(s), "-", "_", -1)
}

type imageManifest []struct {
	RepoTags []string
}

func extractImageManifest(r io.Reader) (imageManifest, error) {
	mtr := tar.NewReader(r)

	var manifest imageManifest

	for {
		mh, err := mtr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if mh.Name == "manifest.json" {
			var mdata bytes.Buffer

			if _, err := io.Copy(&mdata, mtr); err != nil {
				return nil, err
			}

			if err := json.Unmarshal(mdata.Bytes(), &manifest); err != nil {
				return nil, err
			}

			return manifest, nil
		}
	}

	return nil, fmt.Errorf("unable to locate manifest")
}

func systemVolume(v string) bool {
	switch v {
	case "/var/run/docker.sock":
		return true
	case "/var/snap/microk8s/current/docker.sock":
		return true
	}
	return false
}

func (p *Provider) volumeFrom(app, service, v string) string {
	if from := strings.Split(v, ":")[0]; systemVolume(from) {
		return from
	} else if strings.Contains(v, ":") {
		return path.Join("/mnt/volumes", app, "app", from)
	} else {
		return path.Join("/mnt/volumes", app, "service", service, from)
	}
}

func (p *Provider) volumeName(app, v string) string {
	hash := sha256.Sum256([]byte(v))
	name := fmt.Sprintf("%s-%s-%x", p.Rack, app, hash[0:20])
	if len(name) > 63 {
		name = name[0:62]
	}
	return name
}

func (p *Provider) volumeSources(app, service string, vs []string) []string {
	vsh := map[string]bool{}

	for _, v := range vs {
		vsh[p.volumeFrom(app, service, v)] = true
	}

	vsu := []string{}

	for v := range vsh {
		vsu = append(vsu, v)
	}

	sort.Strings(vsu)

	return vsu
}

func volumeTo(v string) (string, error) {
	switch parts := strings.SplitN(v, ":", 2); len(parts) {
	case 1:
		return parts[0], nil
	case 2:
		return parts[1], nil
	default:
		return "", fmt.Errorf("invalid volume %q", v)
	}
}
