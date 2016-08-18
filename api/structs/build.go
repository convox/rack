package structs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"math/rand"
	"os/exec"
	"time"

	"github.com/convox/rack/manifest"
)

type Build struct {
	Id       string `json:"id"`
	App      string `json:"app"`
	Logs     string `json:"logs"`
	Manifest string `json:"manifest"`
	Release  string `json:"release"`

	Status string `json:"status"`
	Reason string `json:"reason"`

	Description string `json:"description"`

	Started time.Time `json:"started"`
	Ended   time.Time `json:"ended"`
}

type Builds []Build

func NewBuild(app string) *Build {
	return &Build{
		App:    app,
		Id:     generateId("B", 10),
		Status: "created",
	}
}

func (b *Build) Export(imageRepo string) ([]byte, error) {

	buf := &bytes.Buffer{}
	gz := gzip.NewWriter(buf)
	tw := tar.NewWriter(gz)

	m, err := manifest.Load([]byte(b.Manifest))
	if err != nil {
		return nil, fmt.Errorf("manifest error: %s", err)
	}

	if len(m.Services) < 1 {
		return nil, fmt.Errorf("no services found to export")
	}

	bjson, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}

	dataHeader := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "builddata.json",
		Mode:     0600,
		Size:     int64(len(bjson)),
	}

	if err := tw.WriteHeader(dataHeader); err != nil {
		return nil, err
	}

	if _, err := tw.Write(bjson); err != nil {
		return nil, err
	}

	for service := range m.Services {

		image, err := exec.Command("docker", "save", fmt.Sprintf("%s:%s.%s", imageRepo, service, b.Id)).Output()
		if err != nil {
			return nil, err
		}

		header := &tar.Header{
			Typeflag: tar.TypeReg,
			Name:     fmt.Sprintf("%s.%s.tar", service, b.Id),
			Mode:     0600,
			Size:     int64(len(image)),
		}

		if err := tw.WriteHeader(header); err != nil {
			return nil, err
		}

		if _, err := tw.Write(image); err != nil {
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), err
}

var idAlphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func generateId(prefix string, size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = idAlphabet[rand.Intn(len(idAlphabet))]
	}
	return prefix + string(b)
}
