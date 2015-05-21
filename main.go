package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"os"
	"strings"
)

var (
	flagPorts     StringSet
	flagBalancers string
)

func init() {
	flag.Var(&flagPorts, "p", "port mapping")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "app: build convox app stack\n\nUsage:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n  app -p 80:3000 -p 443:4000\n")
	}
}

type Port struct {
	Balancer  string
	Container string
}

type StringSet []string

func (ss *StringSet) Set(value string) error {
	*ss = append(*ss, value)
	return nil
}

func (ss *StringSet) String() string {
	return fmt.Sprintf("[%s]", strings.Join(*ss, ","))
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

	ports := parsePorts(flagPorts)

	params := map[string]interface{}{
		"Ports": ports,
	}

	if len(ports) > 0 {
		params["HasPorts"] = true
		params["FirstContainerPort"] = ports[0].Container
	}

	data, err := buildTemplate("formation", "app", params)

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

func parsePorts(ss StringSet) []Port {
	pp := make([]Port, len(ss))

	for i, s := range ss {
		sp := strings.SplitN(s, ":", 2)

		if len(sp) != 2 {
			die(fmt.Errorf("error: must specify balancer:container mapping\n"))
		}

		pp[i] = Port{Balancer: sp[0], Container: sp[1]}
	}

	return pp
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
		"array": func(ss []string) template.HTML {
			as := make([]string, len(ss))
			for i, s := range ss {
				as[i] = fmt.Sprintf("%q", s)
			}
			return template.HTML(strings.Join(as, ", "))
		},
		"join": func(s []string, t string) string {
			return strings.Join(s, t)
		},
		"listeners": func(pp []Port) template.HTML {
			ss := make([]string, len(pp))

			for i, p := range pp {
				ss[i] = fmt.Sprintf(`{ "Protocol": "TCP", "LoadBalancerPort": "%s", "InstanceProtocol": "TCP", "InstancePort": "%s" }`, p.Balancer, p.Container)
			}

			return template.HTML(strings.Join(ss, ","))
		},
		"ports": func(nn []int) template.HTML {
			as := make([]string, len(nn))
			for i, n := range nn {
				as[i] = fmt.Sprintf("%d", n)
			}
			return template.HTML(strings.Join(as, ","))
		},
		"safe": func(s string) template.HTML {
			return template.HTML(s)
		},
		"securityGroups": func(pp []Port) template.HTML {
			ss := make([]string, len(pp))

			for i, p := range pp {
				ss[i] = fmt.Sprintf(`{ "IpProtocol": "tcp", "FromPort": "%s", "ToPort": "%s", "CidrIp": "0.0.0.0/0" }`, p.Balancer, p.Balancer)
			}

			return template.HTML(strings.Join(ss, ","))
		},
		"upper": func(name string) string {
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
		},
	}
}
