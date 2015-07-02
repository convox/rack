package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

var (
	flagMode string
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	flag.StringVar(&flagMode, "mode", "staging", "deployment mode")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: convox/app [options]\n")
		fmt.Fprintf(os.Stderr, "  expects an optional docker-compose.yml on stdin\n\n")
		fmt.Fprintf(os.Stderr, "options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nexamples:\n")
		fmt.Fprintf(os.Stderr, "  cat docker-compose.yml | docker run -i convox/app -mode staging\n")
	}
}

type ManifestEntry struct {
	Command interface{} `yaml:"command"`
	Links   []string    `yaml:"links"`
	Ports   []string    `yaml:"ports"`
	Volumes []string    `yaml:"volumes"`

	Randoms []string
}

type Manifest map[string]ManifestEntry

type Listener struct {
	Balancer string
	Process  string
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "error: %s\n", err)
	os.Exit(1)
}

func usage() {
	fmt.Fprintf(os.Stderr, `app: build convox app stacks

Options:
  -p <balancer:container>    map a port on the balancer to a container port
`)

	os.Exit(1)
}

func main() {
	flag.Parse()

	var manifest Manifest

	if stat, _ := os.Stdin.Stat(); stat.Mode()&os.ModeCharDevice == 0 {
		man, err := ioutil.ReadAll(os.Stdin)

		if err != nil {
			die(err)
		}

		err = yaml.Unmarshal(man, &manifest)

		if err != nil {
			die(err)
		}
	}

	for i, e := range manifest {
		for _ = range e.Ports {
			e.Randoms = append(e.Randoms, randomPort())
		}
		manifest[i] = e
	}

	data, err := buildTemplate(flagMode, "formation", manifest)

	if err != nil {
		displaySyntaxError(data, err)
		die(err)
	}

	pretty, err := prettyJson(data)

	if err != nil {
		displaySyntaxError(data, err)
		die(err)
	}

	fmt.Println(pretty)
}

func buildTemplate(name, section string, data interface{}) (string, error) {
	tmpl, err := template.New(section).Funcs(templateHelpers()).ParseFiles(fmt.Sprintf("template/%s.tmpl", name))

	if err != nil {
		return "", err
	}

	var formation bytes.Buffer

	err = tmpl.Execute(&formation, data)

	if err != nil {
		return "", err
	}

	return formation.String(), nil
}

func displaySyntaxError(data string, err error) {
	syntax, ok := err.(*json.SyntaxError)

	if !ok {
		fmt.Println(err)
		return
	}

	start, end := strings.LastIndex(data[:syntax.Offset], "\n")+1, len(data)

	if idx := strings.Index(data[start:], "\n"); idx >= 0 {
		end = start + idx
	}

	line, pos := strings.Count(data[:start], "\n"), int(syntax.Offset)-start-1

	fmt.Printf("Error in line %d: %s \n", line, err)
	fmt.Printf("%s\n%s^\n", data[start:end], strings.Repeat(" ", pos))
}

func parseList(list string) []string {
	return strings.Split(list, ",")
}

func parseListeners(list string) []Listener {
	items := parseList(list)

	listeners := make([]Listener, len(items))

	for i, l := range items {
		parts := strings.SplitN(l, ":", 2)

		if len(parts) != 2 {
			die(fmt.Errorf("listeners must be balancer:process pairs"))
		}

		listeners[i] = Listener{Balancer: parts[0], Process: parts[1]}
	}

	return listeners
}

func prettyJson(raw string) (string, error) {
	var parsed map[string]interface{}

	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return "", err
	}

	bp, err := json.MarshalIndent(parsed, "", "  ")

	if err != nil {
		return "", err
	}

	clean := strings.Replace(string(bp), "\n\n", "\n", -1)

	return clean, nil
}

func printLines(data string) {
	lines := strings.Split(data, "\n")

	for i, line := range lines {
		fmt.Printf("%d: %s\n", i, line)
	}
}

func templateHelpers() template.FuncMap {
	return template.FuncMap{
		"command": func(command interface{}) string {
			switch cmd := command.(type) {
			case nil:
				return ""
			case string:
				return cmd
			case []interface{}:
				parts := make([]string, len(cmd))

				for i, c := range cmd {
					parts[i] = c.(string)
				}

				return strings.Join(parts, " ")
			default:
				fmt.Fprintf(os.Stderr, "unexpected type for command: %T\n", cmd)
			}
			return ""
		},
		"ingress": func(m Manifest) template.HTML {
			ls := []string{}

			for ps, entry := range m {
				for _, port := range entry.Ports {
					parts := strings.SplitN(port, ":", 2)

					if len(parts) != 2 {
						continue
					}

					ls = append(ls, fmt.Sprintf(`{ "CidrIp": "0.0.0.0/0", "IpProtocol": "tcp", "FromPort": { "Ref": "%sPort%sBalancer" }, "ToPort": { "Ref": "%sPort%sBalancer" } }`, upperName(ps), parts[0], upperName(ps), parts[0]))
				}
			}

			return template.HTML(strings.Join(ls, ","))
		},
		"listeners": func(m Manifest) template.HTML {
			ls := []string{}

			for ps, entry := range m {
				for _, port := range entry.Ports {
					parts := strings.SplitN(port, ":", 2)

					if len(parts) != 2 {
						continue
					}

					ls = append(ls, fmt.Sprintf(`{ "Protocol": "TCP", "LoadBalancerPort": { "Ref": "%sPort%sBalancer" }, "InstanceProtocol": "TCP", "InstancePort": { "Ref": "%sPort%sHost" } }`, upperName(ps), parts[0], upperName(ps), parts[0]))
				}
			}

			return template.HTML(strings.Join(ls, ","))
		},
		"loadbalancers": func(m Manifest) template.HTML {
			ls := []string{}

			for ps, entry := range m {
				for _, port := range entry.Ports {
					parts := strings.SplitN(port, ":", 2)

					if len(parts) != 2 {
						continue
					}

					ls = append(ls, fmt.Sprintf(`{ "Fn::Join": [ ":", [ { "Ref": "Balancer" }, "%s", "%s" ] ] }`, ps, parts[1]))
				}
			}

			return template.HTML(strings.Join(ls, ","))
		},
		"dkeys": func(m Manifest) template.HTML {
			keys := []string{}

			for k, _ := range m {
				keys = append(keys, fmt.Sprintf(`"%s:%s"`, k, k))
			}

			return template.HTML(strings.Join(keys, ","))
		},
		"keys": func(m Manifest) template.HTML {
			keys := []string{}

			for k, _ := range m {
				keys = append(keys, fmt.Sprintf(`"%s"`, k))
			}

			return template.HTML(strings.Join(keys, ","))
		},
		"links": func(m Manifest) template.HTML {
			links := []string{}

			for ps, _ := range m {
				links = append(links, fmt.Sprintf(`{ "Fn::If": [ "Blank%sService", "%s:%s", { "Ref": "AWS::NoValue" } ] }`, upperName(ps), ps, ps))
			}

			return template.HTML(strings.Join(links, ","))
		},
		"names": func(m Manifest) template.HTML {
			names := []string{}

			for ps, _ := range m {
				names = append(names, fmt.Sprintf(`{ "Fn::If": [ "Blank%sService", "%s", { "Ref": "AWS::NoValue" } ] }`, upperName(ps), ps))
			}

			return template.HTML(strings.Join(names, ","))
		},
		"safe": func(s string) template.HTML {
			return template.HTML(s)
		},
		"split": func(ss string, t string) []string {
			return strings.Split(ss, t)
		},
		"tasks": func(m Manifest) template.HTML {
			ls := []string{}

			for ps, entry := range m {
				mappings := []string{}

				for _, port := range entry.Ports {
					parts := strings.SplitN(port, ":", 2)

					if len(parts) != 2 {
						continue
					}

					mappings = append(mappings, fmt.Sprintf(`{ "Fn::Join": [ ":", [ { "Ref": "%sPort%sHost" }, "%s" ] ] }`, upperName(ps), parts[0], parts[1]))
				}

				links := make([]string, len(entry.Links))

				for i, link := range entry.Links {
					links[i] = fmt.Sprintf(`{ "Fn::If": [ "Blank%sService",
						"%s:%s",
						{ "Ref" : "AWS::NoValue" } ] }`, upperName(link), link, link)
				}

				services := make([]string, len(entry.Links))

				for i, link := range entry.Links {
					services[i] = fmt.Sprintf(`{ "Fn::If": [ "Blank%sService",
						{ "Ref" : "AWS::NoValue" },
						{ "Fn::Join": [ ":", [ { "Ref" : "%sService" }, "%s" ] ] } ] }`, upperName(link), upperName(link), link)
				}

				volumes := []string{}

				for _, volume := range entry.Volumes {
					if strings.HasPrefix(volume, "/var/run/docker.sock") {
						volumes = append(volumes, fmt.Sprintf(`"%s"`, volume))
					}
				}

				ls = append(ls, fmt.Sprintf(`{ "Fn::If": [ "Blank%sService",
				{
					"Name": "%s",
					"Image": { "Ref": "%sImage" },
					"Command": { "Ref": "%sCommand" },
					"Key": { "Ref": "Key" },
					"CPU": { "Ref": "Cpu" },
					"Memory": { "Ref": "Memory" },
					"Environment": {
						"KINESIS": { "Ref": "Kinesis" },
						"PROCESS": "%s"
					},
					"Links": [ %s ],
					"Volumes": [ %s ],
					"Services": [ %s ],
					"PortMappings": [ %s ]
				}, { "Ref" : "AWS::NoValue" } ] }`, upperName(ps), ps, upperName(ps), upperName(ps), ps, strings.Join(links, ","), strings.Join(volumes, ","), strings.Join(services, ","), strings.Join(mappings, ",")))
			}

			return template.HTML(strings.Join(ls, ","))
		},
		"upper": func(name string) string {
			return upperName(name)
		},
		"upperenv": func(name string) string {
			return upperEnv(name)
		},
	}
}

func randomPort() string {
	return strconv.Itoa(rand.Intn(50000) + 5000)
}

var regexpNonAlpha = regexp.MustCompile("[^a-zA-Z]")

func upperEnv(name string) string {
	return strings.ToUpper(regexpNonAlpha.ReplaceAllString(name, "_"))
}

func upperName(name string) string {
	us := strings.ToUpper(name[0:1]) + name[1:]

	for {
		i := strings.Index(us, "-")

		if i == -1 {
			break
		}

		s := us[0:i]

		if len(us) > i+1 {
			s += strings.ToUpper(us[i+1 : i+2])
		}

		if len(us) > i+2 {
			s += us[i+2:]
		}

		us = s
	}

	return us
}

func (m Manifest) FirstPort() string {
	for _, me := range m {
		if len(me.Ports) > 0 {
			return strings.Split(me.Ports[0], ":")[0]
		}
	}

	return ""
}

func (m Manifest) FirstRandom() string {
	for _, me := range m {
		if len(me.Randoms) > 0 {
			return me.Randoms[0]
		}
	}

	return ""
}

func (m Manifest) HasPorts() bool {
	for _, me := range m {
		if len(me.Ports) > 0 {
			return true
		}
	}

	return false
}

func (m Manifest) HasProcesses() bool {
	return len(m) > 0
}

func (me ManifestEntry) HasPorts() bool {
	return len(me.Ports) > 0
}
