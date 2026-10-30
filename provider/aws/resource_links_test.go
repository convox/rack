package aws

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func appDeleteProvider(t *testing.T, link string, requests *[]string) *Provider {
	stack := func(name, tags, outputs string) string {
		return `<member><StackName>` + name + `</StackName><StackStatus>CREATE_COMPLETE</StackStatus><Tags>` + tags + `</Tags><Outputs>` + outputs + `</Outputs></member>`
	}
	tag := func(k, v string) string {
		return `<member><Key>` + k + `</Key><Value>` + v + `</Value></member>`
	}

	app := stack("convox-myapp", tag("System", "convox")+tag("Type", "app")+tag("Rack", "convox")+tag("Name", "myapp"), "")
	syslog := stack("convox-syslog1", tag("System", "convox")+tag("Type", "resource")+tag("Rack", "convox")+tag("Name", "syslog1")+tag("Resource", "syslog"),
		`<member><OutputKey>MyappLink</OutputKey><OutputValue>`+link+`</OutputValue></member>`)
	db := stack("convox-db1", tag("System", "convox")+tag("Type", "resource")+tag("Rack", "convox")+tag("Name", "db1")+tag("Resource", "postgres"), "")

	p := stubProviderHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*requests = append(*requests, string(b))

		switch {
		case strings.Contains(string(b), "Action=DescribeStacks&StackName=convox-myapp"):
			_, _ = fmt.Fprint(w, `<DescribeStacksResponse><DescribeStacksResult><Stacks>`+app+`</Stacks></DescribeStacksResult></DescribeStacksResponse>`)
		case strings.Contains(string(b), "Action=DescribeStacks&"):
			_, _ = fmt.Fprint(w, `<DescribeStacksResponse><DescribeStacksResult><Stacks>`+app+syslog+db+`</Stacks></DescribeStacksResult></DescribeStacksResponse>`)
		default:
			w.WriteHeader(400)
			_, _ = fmt.Fprint(w, `<ErrorResponse><Error><Type>Sender</Type><Code>Denied</Code><Message>denied</Message></Error></ErrorResponse>`)
		}
	}))

	return p
}

func TestAppDeleteLinked(t *testing.T) {
	requests := []string{}
	p := appDeleteProvider(t, "convox-myapp-LogGroup-ABC", &requests)

	require.EqualError(t, p.AppDelete("myapp"), "app is linked to syslog1 resource")
	require.Len(t, requests, 2)
}

func TestAppDeleteStaleLink(t *testing.T) {
	requests := []string{}
	p := appDeleteProvider(t, "convox-gone-LogGroup-ABC", &requests)

	err := p.AppDelete("myapp")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Denied: denied")
	require.Len(t, requests, 3)
	require.Contains(t, requests[2], "Action=DeleteStack")
}
