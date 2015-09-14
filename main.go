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
	flagBalancers string
	flagProcesses string
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "service: create a cloudformation stack for a convox service\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n  service postgres\n  service redis\n")
	}
}

func main() {
	flag.Parse()

	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(0)
	}

	// params := map[string]interface{}{
	//   "App":       nil,
	//   "Processes": buildProcesses(parseList(flagProcesses), parseList(flagBalancers)),
	// }

	service := flag.Args()[0]

	data, err := buildTemplate(service, "service", nil)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error building json template: %s\n", err)
		os.Exit(1)
	}

	pretty, err := prettyJson(data)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error building json template: %s\n", err)
		printLines(data)
		displaySyntaxError(data, err)
		os.Exit(1)
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
	if list == "" {
		return []string{}
	}

	parts := strings.Split(list, ",")

	parsed := make([]string, len(parts))

	for i, p := range parts {
		parsed[i] = strings.TrimSpace(p)
	}

	return parsed
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
		"ports": func(nn []int) template.HTML {
			as := make([]string, len(nn))
			for i, n := range nn {
				as[i] = fmt.Sprintf("%d", n)
			}
			return template.HTML(strings.Join(as, ","))
		},
		"processNames": func(pp []Process) string {
			names := make([]string, len(pp))

			for i, p := range pp {
				names[i] = p.Name
			}

			return strings.Join(names, ",")
		},
		"safe": func(s string) template.HTML {
			return template.HTML(s)
		},
		"upper": func(name string) string {
			return strings.ToUpper(name[0:1]) + name[1:]
		},
	}
}

type StringSet []string

func (ss *StringSet) Set(value string) error {
	*ss = append(*ss, value)
	return nil
}

func (ss *StringSet) String() string {
	return fmt.Sprintf("[%s]", strings.Join(*ss, ","))
}
