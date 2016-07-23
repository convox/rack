package manifest

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Process struct {
	Name string

	Args []string

	service Service
}

func NewProcess(app string, s Service) Process {
	name := fmt.Sprintf("%s-%s", app, s.Name)

	args := []string{}

	args = append(args, "-i")
	args = append(args, "--rm")
	args = append(args, "--name", name)

	if s.Entrypoint != "" {
		args = append(args, "--entrypoint", s.Entrypoint)
	}

	for k, v := range s.Environment {
		if v == "" {
			args = append(args, "-e", fmt.Sprintf("%s", k))
		} else {
			args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
		}
	}

	for _, link := range s.Links {
		args = append(args, linkArgs(link, fmt.Sprintf("%s-%s", app, link))...)
	}

	for _, port := range s.Ports {
		args = append(args, "-p", port.String())
	}

	for _, volume := range s.Volumes {
		if strings.Contains(volume, ":") {
			parts := strings.Split(volume, ":")
			if !filepath.IsAbs(parts[0]) {
				absoluteVolumePath, err := filepath.Abs(parts[0])
				if err != nil {
					fmt.Errorf("There was a problem parsing the volume: %s", volume)
				}
				volume = strings.Join([]string{absoluteVolumePath, parts[1]}, ":")
			}
		}
		args = append(args, "-v", volume)
	}

	args = append(args, s.Tag())
	args = append(args, s.Command...)

	return Process{
		Name:    name,
		Args:    args,
		service: s,
	}
}

func (p *Process) Sync(local, remote string) (*Sync, error) {
	return NewSync(p.Name, local, remote)
}
