package aws

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/pkg/test/awsutil"
	"github.com/stretchr/testify/require"
)

func TestResourceTemplateDefaults(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))

	defaults := map[string]map[string]string{}

	parse := func(name string, data []byte) {
		var tmpl struct {
			Parameters map[string]struct {
				Default interface{}
			}
		}
		require.NoError(t, json.Unmarshal(data, &tmpl), name)

		defaults[name] = map[string]string{}
		for k, p := range tmpl.Parameters {
			if s, ok := p.Default.(string); ok {
				defaults[name][k] = s
			}
		}
	}

	apps, err := filepath.Glob("provider/aws/formation/resource/*.json.tmpl")
	require.NoError(t, err)
	require.NotEmpty(t, apps)

	for _, path := range apps {
		kind := strings.TrimSuffix(filepath.Base(path), ".json.tmpl")
		data, err := formationTemplate("resource/"+kind, map[string]interface{}{"Name": "x", "Tags": map[string]string{}, "ThirdAvailabilityZone": false})
		require.NoError(t, err, kind)
		parse("app/"+kind, data)
	}

	racks, err := filepath.Glob("provider/aws/templates/resource/*.tmpl")
	require.NoError(t, err)
	require.NotEmpty(t, racks)

	for _, path := range racks {
		kind := strings.TrimSuffix(filepath.Base(path), ".tmpl")
		data, err := resourceFormation(kind, nil)
		require.NoError(t, err, kind)
		parse("rack/"+kind, []byte(data))
	}

	t2 := regexp.MustCompile(`^(cache|db)\.t2\.`)
	for name, params := range defaults {
		for k, v := range params {
			require.False(t, t2.MatchString(v), "%s %s defaults to %s", name, k, v)
		}
	}

	for _, tc := range []struct {
		template, param, want string
	}{
		{"app/memcached", "Class", "cache.t3.micro"},
		{"app/redis", "Class", "cache.t3.micro"},
		{"app/valkey", "Class", "cache.t3.micro"},
		{"app/mariadb", "Class", "db.t3.micro"},
		{"app/mysql", "Class", "db.t3.micro"},
		{"app/postgres", "Class", "db.t3.micro"},
		{"app/mariadb", "Version", "11.4"},
		{"app/memcached", "Version", "1.4"},
		{"app/mysql", "Version", "8.4"},
		{"app/postgres", "Version", "17"},
		{"app/redis", "Version", "7.0"},
		{"app/valkey", "Version", "8.1"},
		{"rack/memcached", "InstanceType", "cache.t3.micro"},
		{"rack/redis", "InstanceType", "cache.t3.micro"},
		{"rack/valkey", "InstanceType", "cache.t3.micro"},
		{"rack/mysql", "InstanceType", "db.t3.micro"},
		{"rack/postgres", "InstanceType", "db.t3.micro"},
		{"rack/mysql", "EngineVersion", "8.4"},
		{"rack/postgres", "EngineVersion", "17"},
		{"rack/postgres", "Family", "postgres17"},
		{"rack/redis", "EngineVersion", "7.0"},
		{"rack/valkey", "EngineVersion", "8.1"},
	} {
		require.Equal(t, tc.want, defaults[tc.template][tc.param], "%s %s", tc.template, tc.param)
	}
}

func TestAppTemplateResourceParams(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))

	m, err := manifest.Load([]byte(`resources:
  cache:
    type: redis
services:
  web:
    port: 3000
    resources:
      - cache
`), nil)
	require.NoError(t, err)

	data, err := formationTemplate("app", map[string]interface{}{
		"App":                   "testapp",
		"Manifest":              m,
		"Password":              "testpass",
		"Release":               &structs.Release{Id: "R12345", App: "testapp", Build: "B12345"},
		"ResourceParamsCache":   map[string]string{"Class": "cache.t2.micro"},
		"ResourceTemplateCache": "https://example.org/cache.json",
		"Version":               "test",
	})
	require.NoError(t, err)

	var tmpl struct {
		Resources map[string]struct {
			Properties struct {
				Parameters map[string]interface{}
			}
		}
	}
	require.NoError(t, json.Unmarshal(data, &tmpl))
	require.Equal(t, "cache.t2.micro", tmpl.Resources["ResourceCache"].Properties.Parameters["Class"])
}

func TestSystemResourceCreatePostgresFamily(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))

	missing := awsutil.Cycle{
		Request: awsutil.Request{RequestURI: "/", Body: `Action=DescribeStacks&StackName=convox-mydb&Version=2010-05-15`},
		Response: awsutil.Response{
			StatusCode: 400,
			Body:       `<ErrorResponse><Error><Type>Sender</Type><Code>ValidationError</Code><Message>Stack with id convox-mydb does not exist</Message></Error></ErrorResponse>`,
		},
	}
	ignore := awsutil.Cycle{Request: awsutil.Request{Body: "ignore"}, Response: awsutil.Response{StatusCode: 200}}
	create := awsutil.Cycle{
		Request:  awsutil.Request{RequestURI: "/", Body: `/^Action=CreateStack&/`},
		Response: awsutil.Response{StatusCode: 200, Body: `<CreateStackResponse><CreateStackResult><StackId>arn:aws:cloudformation:us-east-1:123456789012:stack/convox-mydb/1</StackId></CreateStackResult></CreateStackResponse>`},
	}

	for _, tc := range []struct {
		params map[string]string
		family string
	}{
		{map[string]string{"EngineVersion": "16"}, "postgres16"},
		{map[string]string{"EngineVersion": "17.9"}, "postgres17"},
		{map[string]string{"EngineVersion": "12.22-rds.20250220"}, "postgres12"},
		{map[string]string{"EngineVersion": "16", "Family": "postgres14"}, "postgres14"},
		{map[string]string{}, ""},
	} {
		p, bodies := stubProvider(t, missing, ignore, create, ignore)

		_, err := p.SystemResourceCreate("postgres", structs.ResourceCreateOptions{Name: options.String("mydb"), Parameters: tc.params})
		require.NoError(t, err, "%v", tc.params)

		params := map[string]string{}
		for _, b := range *bodies {
			if !strings.HasPrefix(b, "Action=CreateStack&") {
				continue
			}

			q, err := url.ParseQuery(b)
			require.NoError(t, err)

			for i := 1; q.Has(fmt.Sprintf("Parameters.member.%d.ParameterKey", i)); i++ {
				params[q.Get(fmt.Sprintf("Parameters.member.%d.ParameterKey", i))] = q.Get(fmt.Sprintf("Parameters.member.%d.ParameterValue", i))
			}
		}

		require.NotEmpty(t, params, "%v", tc.params)
		require.Equal(t, tc.family, params["Family"], "%v", tc.params)
	}
}
