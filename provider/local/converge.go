package local

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/convox/rack/helpers"
	"github.com/convox/rack/manifest"
	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
	"github.com/pkg/errors"
)

var convergeLock sync.Mutex

func (p *Provider) converge(app string) error {
	convergeLock.Lock()
	defer convergeLock.Unlock()

	log := p.logger("converge").Append("app=%q", app)

	m, r, err := helpers.AppManifest(p, app)
	if err != nil {
		return log.Error(err)
	}

	desired := []container{}

	a, err := p.AppGet(app)
	if err != nil {
		return err
	}

	if !a.Sleep {
		var c []container

		c, err = p.resourceContainers(m.Resources, app, r.Id)
		if err != nil {
			return errors.WithStack(log.Error(err))
		}

		desired = append(desired, c...)

		c, err = p.serviceContainers(m.Services, app, r.Id)
		if err != nil {
			return errors.WithStack(log.Error(err))
		}

		desired = append(desired, c...)
	}

	rc, err := containersByLabels(map[string]string{
		"convox.rack": p.Rack,
		"convox.app":  app,
		"convox.type": "resource",
	})
	if err != nil {
		return errors.WithStack(log.Error(err))
	}

	sc, err := containersByLabels(map[string]string{
		"convox.rack": p.Rack,
		"convox.app":  app,
		"convox.type": "service",
	})
	if err != nil {
		return errors.WithStack(log.Error(err))
	}

	current := append(sc, rc...)

	extra := diffContainers(current, desired)
	needed := diffContainers(desired, current)

	for _, c := range extra {
		if err := p.containerStop(c.Id); err != nil {
			return errors.WithStack(log.Error(err))
		}
	}

	for _, c := range needed {
		if _, err := p.containerStart(c, app, r.Id); err != nil {
			return errors.WithStack(log.Error(err))
		}
	}

	if err := p.route(app); err != nil {
		return errors.WithStack(log.Error(err))
	}

	return log.Success()
}

func (p *Provider) idle() error {
	log := p.logger("idle")

	r, err := p.router.RackGet(p.Rack)
	if err != nil {
		return err
	}

	activity := map[string]time.Time{}

	for _, h := range r.Hosts {
		parts := strings.Split(h.Hostname, ".")

		if len(parts) < 2 {
			continue
		}

		app := parts[len(parts)-1]

		if h.Activity.After(activity[app]) {
			activity[app] = h.Activity
		}
	}

	for app, latest := range activity {
		log.Logf("app=%s latest=%s", app, latest)

		if latest.Before(time.Now().UTC().Add(-60 * time.Minute)) {
			if err := p.AppUpdate(app, structs.AppUpdateOptions{Sleep: options.Bool(true)}); err != nil {
				return err
			}
		}
	}

	return nil
}

var serviceEndpoints = map[string]int{
	"http":  80,
	"https": 443,
}

func routeParts(route string) (string, int, error) {
	parts := strings.SplitN(route, ":", 2)

	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid route: %s", route)
	}

	pi, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, err
	}

	return parts[0], pi, nil
}

func (p *Provider) routeContainers(host string, routes map[string]string, labels map[string]string) error {
	cs, err := containersByLabels(labels)
	if err != nil {
		return err
	}

	if err := p.router.HostCreate(p.Rack, host); err != nil {
		return err
	}

	for source, destination := range routes {
		sproto, sport, err := routeParts(source)
		if err != nil {
			return err
		}

		dproto, dport, err := routeParts(destination)
		if err != nil {
			return err
		}

		e, err := p.router.EndpointGet(p.Rack, host, sport)
		if err != nil {
			e, err = p.router.EndpointCreate(p.Rack, host, sproto, sport)
			if err != nil {
				return err
			}
		}

		targets := map[int][]string{}

		for _, c := range cs {
			for p, t := range c.Listeners {
				if targets[p] == nil {
					targets[p] = []string{}
				}
				targets[p] = append(targets[p], fmt.Sprintf("%s://%s", dproto, t))
			}
		}

		missing := diff(targets[dport], e.Targets)
		extra := diff(e.Targets, targets[dport])

		for _, t := range missing {
			if err := p.router.TargetAdd(p.Rack, host, sport, t); err != nil {
				return err
			}
		}

		for _, t := range extra {
			if err := p.router.TargetRemove(p.Rack, host, sport, t); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Provider) route(app string) error {
	m, _, err := helpers.AppManifest(p, app)
	if err != nil {
		return err
	}

	for _, r := range m.Resources {
		rp, err := resourcePort(r.Type)
		if err != nil {
			return err
		}

		host := fmt.Sprintf("%s.resource.%s", r.Name, app)

		routes := map[string]string{
			fmt.Sprintf("tcp:%d", rp): fmt.Sprintf("tcp:%d", rp),
		}

		err = p.routeContainers(host, routes, map[string]string{
			"convox.rack": p.Rack,
			"convox.app":  app,
			"convox.type": "resource",
			"convox.name": r.Name,
		})
		if err != nil {
			return err
		}
	}

	for _, s := range m.Services {
		host := fmt.Sprintf("%s.%s", s.Name, app)

		routes := map[string]string{
			"http:80":   fmt.Sprintf("%s:%d", s.Port.Scheme, s.Port.Port),
			"https:443": fmt.Sprintf("%s:%d", s.Port.Scheme, s.Port.Port),
		}

		err = p.routeContainers(host, routes, map[string]string{
			"convox.rack": p.Rack,
			"convox.app":  app,
			"convox.type": "service",
			"convox.name": s.Name,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func diffContainers(a, b []container) []container {
	diff := []container{}

	for _, aa := range a {
		found := false

		for _, cc := range b {
			if reflect.DeepEqual(aa.Labels, cc.Labels) {
				found = true
				break
			}
		}

		if !found {
			diff = append(diff, aa)
		}
	}

	return diff
}

func resourcePort(kind string) (int, error) {
	switch kind {
	case "mysql":
		return 3306, nil
	case "postgres":
		return 5432, nil
	case "redis":
		return 6379, nil
	}

	return 0, fmt.Errorf("unknown resource type: %s", kind)
}

func (p *Provider) resourceURL(app, kind, name string) (string, error) {
	switch kind {
	case "mysql":
		return fmt.Sprintf("mysql://mysql:password@%s.resource.%s.%s:3306/app", name, app, p.Rack), nil
	case "postgres":
		return fmt.Sprintf("postgres://postgres:password@%s.resource.%s.%s:5432/app?sslmode=disable", name, app, p.Rack), nil
	case "redis":
		return fmt.Sprintf("redis://%s.resource.%s.%s:6379/0", name, app, p.Rack), nil
	}

	return "", fmt.Errorf("unknown resource type: %s", kind)
}

func (p *Provider) resourceVolumes(app, kind, name string) ([]string, error) {
	switch kind {
	case "mysql":
		return []string{fmt.Sprintf("%s/%s/resource/%s:/var/lib/mysql", p.Volume, app, name)}, nil
	case "postgres":
		return []string{fmt.Sprintf("%s/%s/resource/%s:/var/lib/postgresql/data", p.Volume, app, name)}, nil
	case "redis":
		return []string{}, nil
	}

	return []string{}, fmt.Errorf("unknown resource type: %s", kind)
}

func (p *Provider) resourceContainers(resources manifest.Resources, app, release string) ([]container, error) {
	cs := []container{}

	for _, r := range resources {
		rp, err := resourcePort(r.Type)
		if err != nil {
			return nil, err
		}

		vs, err := p.resourceVolumes(app, r.Type, r.Name)
		if err != nil {
			return nil, err
		}

		hostname := fmt.Sprintf("%s.resource.%s", r.Name, app)

		cs = append(cs, container{
			Name:     fmt.Sprintf("%s.%s.resource.%s", p.Rack, app, r.Name),
			Hostname: hostname,
			// Targets: []containerTarget{
			//   containerTarget{FromScheme: "tcp", FromPort: rp, ToScheme: "tcp", ToPort: rp},
			// },
			Image:   fmt.Sprintf("convox/%s", r.Type),
			Volumes: vs,
			Port:    rp,
			Labels: map[string]string{
				"convox.rack":     p.Rack,
				"convox.version":  p.Version,
				"convox.app":      app,
				"convox.type":     "resource",
				"convox.name":     r.Name,
				"convox.resource": r.Type,
			},
		})
	}

	return cs, nil
}

func (p *Provider) serviceContainers(services manifest.Services, app, release string) ([]container, error) {
	cs := []container{}

	m, r, err := helpers.ReleaseManifest(p, app, release)
	if err != nil {
		return nil, err
	}

	for _, s := range services {
		cmd := []string{}

		if c := strings.TrimSpace(s.Command); c != "" {
			cmd = append(cmd, "sh", "-c", c)
		}

		env, err := m.ServiceEnvironment(s.Name)
		if err != nil {
			return nil, err
		}

		// copy the map so we can hold on to it
		e := map[string]string{}

		for k, v := range env {
			e[k] = v
		}

		// add resources
		for _, sr := range s.Resources {
			for _, r := range m.Resources {
				if r.Name == sr {
					u, err := p.resourceURL(app, r.Type, r.Name)
					if err != nil {
						return nil, err
					}

					e[fmt.Sprintf("%s_URL", strings.ToUpper(sr))] = u
				}
			}
		}

		vv, err := p.serviceVolumes(app, s.Volumes)
		if err != nil {
			return nil, err
		}

		hostname := fmt.Sprintf("%s.%s", s.Name, app)

		for i := 1; i <= s.Scale.Count.Min; i++ {
			c := container{
				Hostname: hostname,
				Name:     fmt.Sprintf("%s.%s.service.%s.%d", p.Rack, app, s.Name, i),
				Image:    fmt.Sprintf("%s/%s/%s:%s", p.Rack, app, s.Name, r.Build),
				Command:  cmd,
				Env:      e,
				Cpu:      s.Scale.Cpu,
				Memory:   s.Scale.Memory,
				Volumes:  vv,
				Port:     s.Port.Port,
				Labels: map[string]string{
					"convox.rack":     p.Rack,
					"convox.version":  p.Version,
					"convox.app":      app,
					"convox.release":  release,
					"convox.type":     "service",
					"convox.name":     s.Name,
					"convox.hostname": hostname,
					"convox.service":  s.Name,
					"convox.port":     strconv.Itoa(s.Port.Port),
					"convox.scheme":   s.Port.Scheme,
				},
			}

			cs = append(cs, c)
		}
	}

	return cs, nil
}
