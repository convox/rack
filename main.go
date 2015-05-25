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
	flagMode      string
	flagBalancers string
	flagProcesses string
	flagListeners string
)

func init() {
	flag.StringVar(&flagMode, "mode", "production", "deployment mode")
	flag.StringVar(&flagBalancers, "balancers", "", "load balancer list")
	flag.StringVar(&flagProcesses, "processes", "", "process list")
	flag.StringVar(&flagListeners, "listeners", "", "links between load balancers and processes")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: convox/app [options]\n\noptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nexamples:\n")
		fmt.Fprintf(os.Stderr, "  docker run convox/app -balancers front -processes web,worker -listeners front:web\n")
		fmt.Fprintf(os.Stderr, "  docker run convox/app -mode staging -processes worker\n")
	}
}

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

	params := map[string]interface{}{
		"Balancers": parseList(flagBalancers),
		"Processes": parseList(flagProcesses),
		"Listeners": parseListeners(flagListeners),
	}

	data, err := buildTemplate(flagMode, "formation", params)

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
		"ingress": func(balancer string, listeners []Listener) template.HTML {
			ls := []string{}

			for _, l := range listeners {
				if l.Balancer == balancer {
					b := upperName(l.Balancer)
					p := upperName(l.Process)
					ls = append(ls, fmt.Sprintf(`{ "CidrIp": "0.0.0.0/0", "IpProtocol": "tcp", "FromPort": { "Ref": "%s%sBalancerPort" }, "ToPort": { "Ref": "%s%sBalancerPort" } }`, b, p, b, p))
				}
			}

			return template.HTML(strings.Join(ls, ","))
		},
		"listeners": func(balancer string, listeners []Listener) template.HTML {
			ls := []string{}

			for _, l := range listeners {
				if l.Balancer == balancer {
					b := upperName(l.Balancer)
					p := upperName(l.Process)
					ls = append(ls, fmt.Sprintf(`{ "Protocol": "TCP", "LoadBalancerPort": { "Ref": "%s%sBalancerPort" }, "InstanceProtocol": "TCP", "InstancePort": { "Ref": "%s%sHostPort" } }`, b, p, b, p))
				}
			}

			return template.HTML(strings.Join(ls, ","))
		},
		"loadbalancers": func(process string, listeners []Listener) template.HTML {
			ls := []string{}

			for _, l := range listeners {
				if l.Process == process {
					b := upperName(l.Balancer)
					p := upperName(l.Process)
					ls = append(ls, fmt.Sprintf(`{ "Fn::Join": [ ":", [ { "Ref": "%sBalancer" }, { "Ref": "%s%sContainerPort" } ] ] }`, b, b, p))
				}
			}

			return template.HTML(strings.Join(ls, ","))
		},
		"portmappings": func(process string, listeners []Listener) template.HTML {
			ls := []string{}

			for _, l := range listeners {
				if l.Process == process {
					b := upperName(l.Balancer)
					p := upperName(l.Process)
					ls = append(ls, fmt.Sprintf(`{ "Fn::Join": [ ":", [ { "Ref": "%s%sHostPort" }, { "Ref": "%s%sContainerPort" } ] ] }`, b, p, b, p))
				}
			}

			return template.HTML(strings.Join(ls, ","))
		},
		"safe": func(s string) template.HTML {
			return template.HTML(s)
		},
		"upper": func(name string) string {
			return upperName(name)
		},
	}
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
