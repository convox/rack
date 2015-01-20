package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
)

type AppParams struct {
	Name    string
	Cluster string
	Vpc     string
	Cidr    string

	Subnets   []AppParamsSubnet
	Processes []AppParamsProcess
}

type AppParamsSubnet struct {
	Name             string
	AvailabilityZone string
	Cidr             string
	RouteTable       string
	Vpc              string
}

type AppParamsProcess struct {
	Name              string
	Process           string
	Count             int
	Vpc               string
	App               string
	Ami               string
	Cluster           string
	AvailabilityZones []string
	UserData          string
}

type UserdataParams struct {
	Process   string
	Env       map[string]string
	Resources []UserdataParamsResource
	Ports     []int
}

type UserdataParamsResource struct {
}

func parseTemplate(name string, object interface{}) (string, error) {
	funcs := template.FuncMap{
		"array": func(ss []string) template.HTML {
			as := make([]string, len(ss))
			for i, s := range ss {
				as[i] = fmt.Sprintf("%q", s)
			}
			return template.HTML(strings.Join(as, ", "))
		},
		"ports": func(nn []int) template.HTML {
			as := make([]string, len(nn))
			for i, n := range nn {
				as[i] = fmt.Sprintf("%d", n)
			}
			return template.HTML(strings.Join(as, ","))
		},
	}

	tmpl, err := template.New(name).Funcs(funcs).ParseFiles(fmt.Sprintf("provider/templates/%s.tmpl", name))

	if err != nil {
		return "", err
	}

	var formation bytes.Buffer

	err = tmpl.Execute(&formation, object)

	if err != nil {
		return "", err
	}

	raw := formation.String()

	var parsed map[string]interface{}

	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return "", err
	}

	bp, err := json.MarshalIndent(parsed, "", "  ")

	if err != nil {
		return "", err
	}

	return string(bp), nil
}
