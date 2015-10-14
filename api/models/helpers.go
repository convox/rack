package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/s3"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func buildEnvironment() string {
	env := []string{
		fmt.Sprintf("AWS_REGION=%s", os.Getenv("AWS_REGION")),
		fmt.Sprintf("AWS_ACCESS=%s", os.Getenv("AWS_ACCESS")),
		fmt.Sprintf("AWS_SECRET=%s", os.Getenv("AWS_SECRET")),
		fmt.Sprintf("GITHUB_TOKEN=%s", os.Getenv("GITHUB_TOKEN")),
	}
	return strings.Join(env, "\n")
}

func cs(s *string, def string) string {
	if s != nil {
		return *s
	} else {
		return def
	}
}

func ct(t *time.Time) time.Time {
	if t != nil {
		return *t
	} else {
		return time.Time{}
	}
}

func coalesce(s *dynamodb.AttributeValue, def string) string {
	if s != nil {
		return *s.S
	} else {
		return def
	}
}

func flattenTags(tags []cloudformation.Tag) map[string]string {
	f := make(map[string]string)

	for _, tag := range tags {
		f[*tag.Key] = *tag.Value
	}

	return f
}

type Template struct {
	Parameters map[string]TemplateParameter
}

type TemplateParameter struct {
	Default     string
	Description string
	Type        string
}

func formationParameters(formation string) (map[string]TemplateParameter, error) {
	var t Template

	err := json.Unmarshal([]byte(formation), &t)

	if err != nil {
		return nil, err
	}

	return t.Parameters, nil
}

var idAlphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func generateId(prefix string, size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = idAlphabet[rand.Intn(len(idAlphabet))]
	}
	return prefix + string(b)
}

func humanStatus(original string) string {
	switch original {
	case "":
		return "new"
	case "CREATE_IN_PROGRESS":
		return "creating"
	case "CREATE_COMPLETE":
		return "running"
	case "DELETE_FAILED":
		return "running"
	case "DELETE_IN_PROGRESS":
		return "deleting"
	case "ROLLBACK_IN_PROGRESS":
		return "rollback"
	case "ROLLBACK_COMPLETE":
		return "failed"
	case "UPDATE_IN_PROGRESS":
		return "updating"
	case "UPDATE_COMPLETE_CLEANUP_IN_PROGRESS":
		return "updating"
	case "UPDATE_COMPLETE":
		return "running"
	case "UPDATE_ROLLBACK_IN_PROGRESS":
		return "rollback"
	case "UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS":
		return "rollback"
	case "UPDATE_ROLLBACK_COMPLETE":
		return "running"
	default:
		fmt.Printf("unknown status: %s\n", original)
		return "unknown"
	}
}

func linkParts(link string) (string, string, error) {
	parts := strings.Split(link, ":")

	switch len(parts) {
	case 1:
		return parts[0], parts[0], nil
	case 2:
		return parts[0], parts[1], nil
	}

	return "", "", fmt.Errorf("invalid link name")
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

	return string(bp), nil
}

func printLines(data string) {
	lines := strings.Split(data, "\n")

	for i, line := range lines {
		fmt.Printf("%d: %s\n", i, line)
	}
}

func s3Delete(bucket, key string) error {
	req := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err := S3().DeleteObject(req)

	return err
}

func s3Get(bucket, key string) ([]byte, error) {
	req := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	res, err := S3().GetObject(req)

	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(res.Body)
}

func S3Put(bucket, key string, data []byte, public bool) error {
	req := &s3.PutObjectInput{
		Body:          bytes.NewReader(data),
		Bucket:        aws.String(bucket),
		ContentLength: aws.Int64(int64(len(data))),
		Key:           aws.String(key),
	}

	if public {
		req.ACL = aws.String("public-read")
	}

	_, err := S3().PutObject(req)

	return err
}

func S3PutFile(bucket, key string, f io.ReadSeeker, public bool) error {
	// seek to end of f to determine length, then seek back to beginning for upload
	l, err := f.Seek(0, 2)

	if err != nil {
		return err
	}

	_, err = f.Seek(0, 0)

	if err != nil {
		return err
	}

	req := &s3.PutObjectInput{
		Body:          f,
		Bucket:        aws.String(bucket),
		ContentLength: aws.Int64(l),
		Key:           aws.String(key),
	}

	if public {
		req.ACL = aws.String("public-read")
	}

	_, err = S3().PutObject(req)

	if err != nil {
		return err
	}

	// seek back to beginning in case something else needs to read f
	_, err = f.Seek(0, 0)

	return err
}

func stackParameters(stack *cloudformation.Stack) map[string]string {
	parameters := make(map[string]string)

	for _, parameter := range stack.Parameters {
		parameters[*parameter.ParameterKey] = *parameter.ParameterValue
	}

	return parameters
}

func stackOutputs(stack *cloudformation.Stack) map[string]string {
	outputs := make(map[string]string)

	for _, output := range stack.Outputs {
		outputs[*output.OutputKey] = *output.OutputValue
	}

	return outputs
}

func stackTags(stack *cloudformation.Stack) map[string]string {
	tags := make(map[string]string)

	for _, tag := range stack.Tags {
		tags[*tag.Key] = *tag.Value
	}

	return tags
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
		"safe": func(s string) template.HTML {
			return template.HTML(s)
		},
		"upper": func(s string) string {
			return UpperName(s)
		},
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
		"entry_loadbalancers": func(entry ManifestEntry, ps string) template.HTML {
			ls := []string{}

			for _, port := range entry.Ports {
				parts := strings.SplitN(port, ":", 2)

				if len(parts) != 2 {
					continue
				}

				ls = append(ls, fmt.Sprintf(`{ "Fn::Join": [ ":", [ { "Ref": "Balancer" }, "%s", "%s" ] ] }`, ps, parts[1]))
			}

			return template.HTML(strings.Join(ls, ","))
		},
		"entry_task": func(entry ManifestEntry, ps string) template.HTML {
			mappings := []string{}

			for _, port := range entry.Ports {
				parts := strings.SplitN(port, ":", 2)

				switch len(parts) {
				case 1:
					mappings = append(mappings, fmt.Sprintf(`{ "Fn::Join": [ ":", [ { "Ref": "%sPort%sHost" }, "%s" ] ] }`, UpperName(ps), parts[0], parts[0]))
				case 2:
					mappings = append(mappings, fmt.Sprintf(`{ "Fn::Join": [ ":", [ { "Ref": "%sPort%sHost" }, "%s" ] ] }`, UpperName(ps), parts[0], parts[1]))
				}
			}

			envs := make([]string, 0)
			envs = append(envs, fmt.Sprintf("\"PROCESS\": \"%s\"", ps))

			for _, env := range entry.Env {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					envs = append(envs, fmt.Sprintf("\"%s\": \"%s\"", parts[0], parts[1]))
				}
			}

			links := make([]string, len(entry.Links))

			for i, link := range entry.Links {
				name, _, err := linkParts(link)

				if err != nil {
					continue
				}

				// Don't define any links for now, as they won't work with one TaskDefinition per process
				links[i] = fmt.Sprintf(`{ "Fn::If": [ "Blank%sService", { "Ref" : "AWS::NoValue" }, { "Ref" : "AWS::NoValue" } ] }`, UpperName(name))
			}

			services := make([]string, len(entry.Links))

			for i, link := range entry.Links {
				name, _, err := linkParts(link)

				if err != nil {
					continue
				}

				services[i] = fmt.Sprintf(`{ "Fn::If": [ "Blank%sService", { "Ref" : "AWS::NoValue" }, { "Fn::Join": [ ":", [ { "Ref" : "%sService" }, "%s" ] ] } ] }`, UpperName(name), UpperName(name), name)
			}

			volumes := []string{}

			for _, volume := range entry.Volumes {
				if strings.HasPrefix(volume, "/var/run/docker.sock") {
					volumes = append(volumes, fmt.Sprintf(`"%s"`, volume))
				}
			}

			l := fmt.Sprintf(`{ "Fn::If": [ "Blank%sService",
			{
				"Name": "%s",
				"Image": { "Ref": "%sImage" },
				"Command": { "Ref": "%sCommand" },
				"Memory": { "Ref": "%sMemory" },
				"Environment": {
					"KINESIS": { "Ref": "Kinesis" },
					%s
				},
				"Links": [ %s ],
				"Volumes": [ %s ],
				"Services": [ %s ],
				"PortMappings": [ %s ]
			}, { "Ref" : "AWS::NoValue" } ] }`, UpperName(ps), ps, UpperName(ps), UpperName(ps), UpperName(ps), strings.Join(envs, ","), strings.Join(links, ","), strings.Join(volumes, ","), strings.Join(services, ","), strings.Join(mappings, ","))

			return template.HTML(l)
		},
		"ingress": func(m Manifest) template.HTML {
			ls := []string{}

			for _, entry := range m {
				for _, port := range entry.Ports {
					parts := strings.SplitN(port, ":", 2)

					if len(parts) != 2 {
						continue
					}

					ls = append(ls, fmt.Sprintf(`{ "CidrIp": "0.0.0.0/0", "IpProtocol": "tcp", "FromPort": { "Ref": "%sPort%sBalancer" }, "ToPort": { "Ref": "%sPort%sBalancer" } }`, UpperName(entry.Name), parts[0], UpperName(entry.Name), parts[0]))
				}
			}

			return template.HTML(strings.Join(ls, ","))
		},
		"listeners": func(m Manifest) template.HTML {
			ls := []string{}

			for _, entry := range m {
				for _, port := range entry.ExternalPorts() {
					parts := strings.SplitN(port, ":", 2)

					if len(parts) != 2 {
						continue
					}

					l := fmt.Sprintf(`{ "Fn::If": [ "Blank%sPort%sCertificate",
					{
						"Protocol": "TCP",
						"LoadBalancerPort": {
							"Ref": "%sPort%sBalancer" },
							"InstanceProtocol": "TCP",
							"InstancePort": { "Ref": "%sPort%sHost" }
					},
					{
						"Protocol": "SSL",
						"LoadBalancerPort": {
							"Ref": "%sPort%sBalancer" },
							"InstanceProtocol": "TCP",
							"InstancePort": { "Ref": "%sPort%sHost" },
							"SSLCertificateId": { "Ref": "%sPort%sCertificate" }
					} ] }`, UpperName(entry.Name), parts[0], UpperName(entry.Name), parts[0], UpperName(entry.Name), parts[0], UpperName(entry.Name), parts[0], UpperName(entry.Name), parts[0], UpperName(entry.Name), parts[0])

					ls = append(ls, l)
				}
			}

			if len(ls) == 0 {
				ls = append(ls, `{ "Protocol": "TCP", "LoadBalancerPort": "80", "InstanceProtocol": "TCP", "InstancePort": "80" }`)
			}

			return template.HTML(strings.Join(ls, ","))
		},
		"split": func(ss string, t string) []string {
			return strings.Split(ss, t)
		},
		"firstcheck": func(m Manifest) template.HTML {
			for _, me := range m {
				if len(me.Ports) > 0 {
					parts := strings.Split(me.Ports[0], ":")
					port := parts[0]
					return template.HTML(fmt.Sprintf(`{ "Fn::Join": [ ":", [ "TCP", { "Ref": "%sPort%sHost" } ] ] }`, UpperName(me.Name), port))
				}
			}
			return `"TCP:80"`
		},
	}
}

func DashName(name string) string {
	// Myapp -> myapp; MyApp -> my-app
	re := regexp.MustCompile("([a-z])([A-Z])") // lower case letter followed by upper case

	k := re.ReplaceAllString(name, "${1}-${2}")
	return strings.ToLower(k)
}

func UpperName(name string) string {
	// myapp -> Myapp; my-app -> MyApp
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
