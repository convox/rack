package aws

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/pkg/test/awsutil"
	"github.com/stretchr/testify/require"
)

const resourceUpdateLegacyURL = "http://notifications.example.org/sns?endpoint=https%3A%2F%2Fwww.example.com"

func resourceUpdateCycles(name, kind, stored string) []awsutil.Cycle {
	stack := awsutil.Cycle{
		Request: awsutil.Request{RequestURI: "/", Body: fmt.Sprintf("Action=DescribeStacks&StackName=convox-%s&Version=2010-05-15", name)},
		Response: awsutil.Response{StatusCode: 200, Body: `<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
	<DescribeStacksResult>
		<Stacks>
			<member>
				<StackName>convox-` + name + `</StackName>
				<StackStatus>UPDATE_COMPLETE</StackStatus>
				<Parameters>
					<member><ParameterKey>NotificationTopic</ParameterKey><ParameterValue>convox-notifications</ParameterValue></member>
					<member><ParameterKey>PermissionsBoundary</ParameterKey><ParameterValue></ParameterValue></member>
					<member><ParameterKey>Url</ParameterKey><ParameterValue>` + stored + `</ParameterValue></member>
				</Parameters>
				<Tags>
					<member><Key>Name</Key><Value>` + name + `</Value></member>
					<member><Key>Rack</Key><Value>convox</Value></member>
					<member><Key>Resource</Key><Value>` + kind + `</Value></member>
					<member><Key>System</Key><Value>convox</Value></member>
					<member><Key>Type</Key><Value>resource</Value></member>
				</Tags>
			</member>
		</Stacks>
	</DescribeStacksResult>
</DescribeStacksResponse>`},
	}

	rack := awsutil.Cycle{
		Request: awsutil.Request{RequestURI: "/", Body: "Action=DescribeStacks&StackName=convox&Version=2010-05-15"},
		Response: awsutil.Response{StatusCode: 200, Body: `<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
	<DescribeStacksResult>
		<Stacks>
			<member>
				<StackName>convox</StackName>
				<StackStatus>UPDATE_COMPLETE</StackStatus>
			</member>
		</Stacks>
	</DescribeStacksResult>
</DescribeStacksResponse>`},
	}

	settings := awsutil.Cycle{
		Request:  awsutil.Request{RequestURI: "/", Body: "Action=ListStackResources&StackName=convox&Version=2010-05-15"},
		Response: awsutil.Response{StatusCode: 200, Body: buildStuckStackResource("Settings", "convox-settings")},
	}

	update := awsutil.Cycle{
		Request:  awsutil.Request{RequestURI: "/", Body: "/^Action=UpdateStack&/"},
		Response: awsutil.Response{StatusCode: 200, Body: `<UpdateStackResponse><UpdateStackResult><StackId>arn:aws:cloudformation:us-test-1:123456789012:stack/convox-` + name + `/1</StackId></UpdateStackResult></UpdateStackResponse>`},
	}

	cycles := []awsutil.Cycle{stack}
	if kind == "syslog" {
		cycles = append(cycles, stack)
	}

	return append(cycles, rack, stack, settings, awsutil.Cycle{Request: awsutil.Request{Body: "ignore"}, Response: awsutil.Response{StatusCode: 200}}, update)
}

func resourceUpdateURL(t *testing.T, bodies []string) string {
	t.Helper()

	for _, b := range bodies {
		if !strings.HasPrefix(b, "Action=UpdateStack&") {
			continue
		}

		q, err := url.ParseQuery(b)
		require.NoError(t, err)

		for i := 1; q.Has(fmt.Sprintf("Parameters.member.%d.ParameterKey", i)); i++ {
			if q.Get(fmt.Sprintf("Parameters.member.%d.ParameterKey", i)) == "Url" {
				require.False(t, q.Has(fmt.Sprintf("Parameters.member.%d.UsePreviousValue", i)))
				return q.Get(fmt.Sprintf("Parameters.member.%d.ParameterValue", i))
			}
		}
	}

	return ""
}

func TestSystemResourceUpdateWebhookUrl(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))

	console := "https://console.example.com/racks/r/events"

	for _, tc := range []struct {
		name   string
		hook   string
		stored string
		params map[string]string
		url    string
		err    string
	}{
		{"new url", "myhook", "https://example.com/old", map[string]string{"Url": "https://example.com/new"}, "https://example.com/new", ""},
		{"new url over legacy", "myhook", resourceUpdateLegacyURL, map[string]string{"Url": "https://example.com/new"}, "https://example.com/new", ""},
		{"bare, nil", "myhook", "https://example.com/old", nil, "https://example.com/old", ""},
		{"bare, empty map", "myhook", "https://example.com/old", map[string]string{}, "https://example.com/old", ""},
		{"legacy, bare", "myhook", resourceUpdateLegacyURL, map[string]string{}, "https://www.example.com", ""},
		{"empty url", "myhook", "https://example.com/old", map[string]string{"Url": ""}, "", "must specify a URL"},
		{"bad scheme", "myhook", "https://example.com/old", map[string]string{"Url": "ftp://example.com/x"}, "", "invalid URL scheme: ftp. Allowed schemes are: http, https"},
		{"console hook, new url", "console-v1-abc", console, map[string]string{"Url": "https://example.com/new"}, "", "webhook console-v1-abc is managed by Console and its Url cannot be changed"},
		{"console hook, bare", "console-v1-abc", console, map[string]string{}, console, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, bodies := stubProvider(t, resourceUpdateCycles(tc.hook, "webhook", tc.stored)...)

			_, err := p.SystemResourceUpdate(tc.hook, structs.ResourceUpdateOptions{Parameters: tc.params})
			if tc.err == "" {
				require.NoError(t, err)
				require.Equal(t, tc.url, resourceUpdateURL(t, *bodies))
				return
			}

			require.EqualError(t, err, tc.err)
			for _, b := range *bodies {
				require.False(t, strings.HasPrefix(b, "Action=UpdateStack&"))
			}
		})
	}
}

func TestSystemResourceUpdateSyslogUrl(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))

	p, bodies := stubProvider(t, resourceUpdateCycles("mysyslog", "syslog", "tcp+tls://logs.example.com:12346")...)

	_, err := p.SystemResourceUpdate("mysyslog", structs.ResourceUpdateOptions{Parameters: map[string]string{"Url": "tcp+tls://logs.example.com:12347"}})
	require.NoError(t, err)

	require.Equal(t, "tcp+tls://logs.example.com:12347", resourceUpdateURL(t, *bodies))
}

func TestSystemResourceCreateUrlValidation(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))

	for _, tc := range []struct {
		kind   string
		name   string
		params map[string]string
		err    string
	}{
		{"webhook", "myhook", nil, "must specify a URL"},
		{"webhook", "myhook", map[string]string{"Url": "ftp://example.com/x"}, "invalid URL scheme: ftp. Allowed schemes are: http, https"},
		{"syslog", "mysyslog", map[string]string{"Url": "https://example.com"}, "invalid URL scheme: https. Allowed schemes are: tcp, tcp+tls, udp"},
	} {
		missing := awsutil.Cycle{
			Request: awsutil.Request{RequestURI: "/", Body: fmt.Sprintf("Action=DescribeStacks&StackName=convox-%s&Version=2010-05-15", tc.name)},
			Response: awsutil.Response{
				StatusCode: 400,
				Body:       `<ErrorResponse><Error><Type>Sender</Type><Code>ValidationError</Code><Message>Stack with id convox-` + tc.name + ` does not exist</Message></Error></ErrorResponse>`,
			},
		}

		p, bodies := stubProvider(t, missing)

		_, err := p.SystemResourceCreate(tc.kind, structs.ResourceCreateOptions{Name: options.String(tc.name), Parameters: tc.params})
		require.EqualError(t, err, tc.err)
		require.Len(t, *bodies, 1)
	}
}

func TestSystemResourceCreateConsoleWebhook(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))

	missing := awsutil.Cycle{
		Request: awsutil.Request{RequestURI: "/", Body: "Action=DescribeStacks&StackName=convox-console-v1-abc&Version=2010-05-15"},
		Response: awsutil.Response{
			StatusCode: 400,
			Body:       `<ErrorResponse><Error><Type>Sender</Type><Code>ValidationError</Code><Message>Stack with id convox-console-v1-abc does not exist</Message></Error></ErrorResponse>`,
		},
	}
	ignore := awsutil.Cycle{Request: awsutil.Request{Body: "ignore"}, Response: awsutil.Response{StatusCode: 200}}
	create := awsutil.Cycle{
		Request:  awsutil.Request{RequestURI: "/", Body: `/^Action=CreateStack&.*ParameterValue=https%3A%2F%2Fconsole.example.com%2Fracks%2Fr%2Fevents&/`},
		Response: awsutil.Response{StatusCode: 200, Body: `<CreateStackResponse><CreateStackResult><StackId>arn:aws:cloudformation:us-test-1:123456789012:stack/convox-console-v1-abc/1</StackId></CreateStackResult></CreateStackResponse>`},
	}

	p, _ := stubProvider(t, missing, ignore, create, ignore)

	_, err := p.SystemResourceCreate("webhook", structs.ResourceCreateOptions{Name: options.String("console-v1-abc"), Parameters: map[string]string{"Url": "https://console.example.com/racks/r/events"}})
	require.NoError(t, err)
}
