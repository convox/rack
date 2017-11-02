package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/rack/manifest"
	"github.com/convox/rack/manifest1"
	shellquote "github.com/kballard/go-shellquote"
	"gopkg.in/urfave/cli.v1"
	yaml "gopkg.in/yaml.v2"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "init",
		Description: "initialize an app for local development",
		Usage:       "[directory]",
		Action:      cmdInit,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "boilerplate",
				Usage: "generate a simple boilerplate app",
			},
		},
	})
}

func cmdInit(c *cli.Context) error {
	if _, err := os.Stat("convox.yml"); err == nil {
		return fmt.Errorf("convox.yml already exists")
	}

	m, err := manifest1.LoadFile("docker-compose.yml")
	if err != nil {
		return err
	}

	mNew, report, err := ManifestConvert(m)
	if err != nil {
		return err
	}

	ymNew, err := yaml.Marshal(mNew)
	if err != nil {
		return err
	}

	report.Print()

	if report.Success {
		err = ioutil.WriteFile("convox.yml", ymNew, 0644)
		if err != nil {
			return err
		}
		fmt.Printf("wrote manifest: convox.yml\n")
	} else {
		return fmt.Errorf("address FAIL messages and try again")
	}

	return nil
}

type Report struct {
	Success  bool
	Messages []string
}

func (r *Report) Append(m string) {
	r.Messages = append(r.Messages, m)
}

func (r Report) Print() {
	sw := *stdcli.DefaultWriter
	for _, message := range r.Messages {
		sw.Writef(message)
	}
}

func ManifestConvert(mOld *manifest1.Manifest) (*manifest.Manifest, Report, error) {
	report := Report{Success: true}

	resources := make(manifest.Resources, 0)
	services := manifest.Services{}
	timers := make(manifest.Timers, 0)

	sk := make([]string, 0)
	for k, _ := range mOld.Services {
		sk = append(sk, k)
	}
	sort.Strings(sk)

	for _, k := range sk {
		service := mOld.Services[k]

		// resources
		serviceResources := []string{}
		if resourceService(service) {
			t := ""
			switch service.Image {
			case "convox/mysql":
				t = "mysql"
			case "convox/postgres":
				t = "postgres"
			case "convox/redis":
				t = "redis"
			case "convox/rabbitmq":
				t = "rabbitmq"
			case "convox/elasticsearch":
				t = "elasticsearch"
			default:
				return nil, report, fmt.Errorf("%s is not a recognized resource image", service.Image)
			}
			r := manifest.Resource{
				Name: service.Name,
				Type: t,
			}
			resources = append(resources, r)

			report.Append(fmt.Sprintf("INFO: service <service>%s</service> has been migrated to a resource\n", service.Name))
			continue
		}

		// build
		b := manifest.ServiceBuild{
			Path: service.Build.Context,
		}

		// build args
		if len(service.Build.Args) > 0 {
			report.Append(fmt.Sprintf("<fail>FAIL</fail>: <service>%s</service> build args not migrated to convox.yml, use ARG in your Dockerfile instead\n", service.Name))
			report.Success = false
		}

		// build dockerfile
		if service.Build.Dockerfile != "" {
			report.Append(fmt.Sprintf("<fail>FAIL</fail>: <service>%s</service> \"dockerfile\" key is not supported in convox.yml, file must be named \"Dockerfile\"\n", service.Name))
			report.Success = false
		}

		// command
		var cmd string
		if len(service.Command.Array) > 0 {
			cmd = shellquote.Join(service.Command.Array...)
		} else {
			cmd = service.Command.String
		}

		// entrypoint
		if service.Entrypoint != "" {
			report.Append(fmt.Sprintf("<fail>FAIL</fail>: <service>%s</service> \"entrypoint\" key not supported in convox.yml, use ENTRYPOINT in Dockerfile instead\n", service.Name))
			report.Success = false
		}

		// environment
		env := []string{}
		for _, eItem := range service.Environment {
			if eItem.Needed {
				env = append(env, eItem.Name)
			} else {
				env = append(env, fmt.Sprintf("%s=%s", eItem.Name, eItem.Value))
			}
		}

		// convox.agent
		if service.IsAgent() {
			report.Append(fmt.Sprintf("INFO: <service>%s</service> - running as an agent is not supported\n", service.Name))
		}

		// convox.balancer
		if (len(service.Ports) > 0) && !service.HasBalancer() {
			report.Append(fmt.Sprintf("INFO: <service>%s</service> - disabling balancers is not supported\n", service.Name))
		}

		// convox.cron
		for k, v := range service.LabelsByPrefix("convox.cron") {
			timer := manifest.Timer{}
			ks := strings.Split(k, ".")
			tokens := strings.Fields(v)
			timer.Name = ks[len(ks)-1]
			timer.Command = strings.Join(tokens[5:], " ")
			timer.Schedule = strings.Join(tokens[0:5], " ")
			timer.Service = service.Name
			timers = append(timers, timer)
		}

		// convox.draining.timeout
		if len(service.LabelsByPrefix("convox.draining.timeout")) > 0 {
			report.Append(fmt.Sprintf("INFO: <service>%s</service> - setting draning timeout is not supported\n", service.Name))
		}

		// convox.environment.secure
		if len(service.LabelsByPrefix("convox.environment.secure")) > 0 {
			report.Append(fmt.Sprintf("INFO: <service>%s</service> - setting secure environment is not necessary\n", service.Name))
		}

		// convox.health.path
		// convox.health.timeout
		health := manifest.ServiceHealth{}
		if balancer := mOld.GetBalancer(service.Name); balancer != nil {
			timeout, err := strconv.Atoi(balancer.HealthTimeout())
			if err != nil {
				return nil, report, err
			}
			health.Path = balancer.HealthPath()
			health.Timeout = timeout
		}

		// convox.health.port
		if len(service.LabelsByPrefix("convox.health.port")) > 0 {
			report.Append(fmt.Sprintf("INFO: <service>%s</service> - setting health check port is not necessary\n", service.Name))
		}

		// convox.health.threshold.healthy
		// convox.helath.threshold.unhealthy
		if len(service.LabelsByPrefix("convox.health.threshold")) > 0 {
			report.Append(fmt.Sprintf("INFO: <service>%s</service> - setting health check thresholds is not supported\n", service.Name))
		}

		// convox.idle.timeout
		if len(service.LabelsByPrefix("convox.idle.timeout")) > 0 {
			report.Append(fmt.Sprintf("INFO: <service>%s</service> - setting idle timeout is not supported\n", service.Name))
		}

		// convox.port..protocol
		// convox.port..proxy
		// convox.port..secure
		if len(service.LabelsByPrefix("convox.idle.timeout")) > 0 {
			report.Append(fmt.Sprintf("INFO: <service>%s</service> - configuring balancer via convox.port labels is not supported\n", service.Name))
		}

		// convox.start.shift
		if len(service.LabelsByPrefix("convox.start.shift")) > 0 {
			report.Append(fmt.Sprintf("<fail>FAIL</fail>: <service>%s</service> - port shifting is not supported, use internal hostnames instead\n", service.Name))
			report.Success = false
		}

		// links
		for _, link := range service.Links {
			resource := false
			for _, sOld := range mOld.Services {
				if (sOld.Name == link) && resourceService(sOld) {
					serviceResources = append(serviceResources, link)
					resource = true
					break
				}
			}
			if !resource {
				report.Append(fmt.Sprintf("INFO: <service>%s</service> - environment variables not generated for linked service <service>%s</service>, use internal URL https://%s.<app name>.convox instead\n", service.Name, link, link))
			}
		}

		// mem_limit
		mb := service.Memory / (1024 * 1024) // bytes to Megabytes
		scale := manifest.ServiceScale{
			Memory: int(mb),
		}

		// ports
		p := manifest.ServicePort{}

		for _, port := range service.Ports {
			if port.Protocol == "udp" {
				report.Append(fmt.Sprintf("INFO: <service>%s</service> - UDP ports are not supported\n", service.Name))
				continue
			}

			switch port.Balancer {
			case 80, 443:
			default:
				report.Append(fmt.Sprintf("INFO: <service>%s</service> - only HTTP ports supported\n", service.Name))
				continue
			}

			p.Port = port.Container
			p.Scheme = "http"

			if service.Labels[fmt.Sprintf("convox.port.%d.secure", port.Balancer)] == "true" {
				p.Scheme = "https"
			}
		}

		// privileged
		if service.Privileged {
			report.Append(fmt.Sprintf("INFO: <service>%s</service> - privileged mode not supported\n", service.Name))
		}

		s := manifest.Service{
			Name:        k,
			Build:       b,
			Command:     cmd,
			Environment: env,
			Health:      health,
			Image:       service.Image,
			Port:        p,
			Resources:   serviceResources,
			Scale:       scale,
			Volumes:     service.Volumes,
		}
		services = append(services, s)
	}

	if mOld.Networks != nil {
		report.Append(fmt.Sprintf("INFO: custom networks not supported, use service hostnames instead\n"))
	}

	m := manifest.Manifest{
		Resources: resources,
		Services:  services,
		Timers:    timers,
	}

	err := m.ApplyDefaults()
	if err != nil {
		return nil, report, err
	}

	return &m, report, nil
}

func resourceService(service manifest1.Service) bool {
	resourceImages := []string{
		"convox/mysql",
		"convox/postgres",
		"convox/redis",
		"convox/rabbitmq",
		"convox/elasticsearch",
	}

	for _, image := range resourceImages {
		if service.Image == image {
			return true
		}
	}

	return false
}
