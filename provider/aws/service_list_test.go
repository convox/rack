package aws

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceListCertificate(t *testing.T) {
	arn := "arn:aws:acm:us-test-1:123456789012:certificate/1ad574bd-eeb0-466e-b961-74ec8b405093"
	item := `{"id":{"S":"R1"},"app":{"S":"myapp"},"created":{"S":"20160404.143416.178278576"},"manifest":{"S":"services:\n  web:\n    port: 3000\n"}}`

	member := func(k, v string) string {
		return `<member><Key>` + k + `</Key><Value>` + v + `</Value></member>`
	}
	output := func(k, v string) string {
		return `<member><OutputKey>` + k + `</OutputKey><OutputValue>` + v + `</OutputValue></member>`
	}

	ops := map[string]int{}

	p := stubProviderHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body := string(b)

		op := r.Header.Get("X-Amz-Target")
		if op == "" && strings.HasPrefix(body, "Action=") {
			op = strings.TrimPrefix(strings.Split(body, "&")[0], "Action=")
		}
		if op == "" {
			op = r.Method + " " + r.URL.Path
		}
		ops[op]++

		switch op {
		case "DescribeStacks":
			_, _ = fmt.Fprint(w, `<DescribeStacksResponse><DescribeStacksResult><Stacks><member><StackName>convox-myapp</StackName><StackStatus>UPDATE_COMPLETE</StackStatus>`+
				`<Tags>`+member("System", "convox")+member("Type", "app")+member("Rack", "convox")+member("Name", "myapp")+member("Generation", "2")+`</Tags>`+
				`<Outputs>`+output("Release", "R1")+output("ServiceWebEndpoint", "web.example.org")+output("ServiceWebCertificate", arn)+`</Outputs>`+
				`<Parameters><member><ParameterKey>WebFormation</ParameterKey><ParameterValue>1,256,512</ParameterValue></member></Parameters>`+
				`</member></Stacks></DescribeStacksResult></DescribeStacksResponse>`)
		case "ListStackResources":
			_, _ = fmt.Fprint(w, `<ListStackResourcesResponse><ListStackResourcesResult><StackResourceSummaries>`+
				`<member><LogicalResourceId>Settings</LogicalResourceId><PhysicalResourceId>settings</PhysicalResourceId></member>`+
				`<member><LogicalResourceId>EncryptionKey</LogicalResourceId><PhysicalResourceId>arn:aws:kms:us-test-1:123456789012:key/k</PhysicalResourceId></member>`+
				`</StackResourceSummaries></ListStackResourcesResult></ListStackResourcesResponse>`)
		case "DynamoDB_20120810.GetItem":
			_, _ = fmt.Fprint(w, `{"Item":`+item+`}`)
		case "DynamoDB_20120810.Query":
			_, _ = fmt.Fprint(w, `{"Count":1,"Items":[`+item+`]}`)
		case "GET /settings/releases/R1/env":
			_, _ = fmt.Fprint(w, "FOO=bar\n")
		case "CertificateManager.ListTagsForCertificate":
			_, _ = fmt.Fprint(w, `{"Tags":[]}`)
		case "CertificateManager.DescribeCertificate":
			_, _ = fmt.Fprint(w, `{"Certificate":{"CertificateArn":"`+arn+`","Status":"ISSUED"}}`)
		default:
			w.WriteHeader(400)
		}
	}))

	p.DynamoReleases = "convox-releases"

	ss, err := p.ServiceList("myapp")
	require.NoError(t, err)
	require.Len(t, ss, 1)
	require.Len(t, ss[0].Ports, 2)
	assert.Equal(t, "acm-74ec8b405093", ss[0].Ports[1].Certificate)

	assert.Equal(t, 1, ops["DynamoDB_20120810.GetItem"])
	assert.Equal(t, 1, ops["CertificateManager.DescribeCertificate"])
	assert.Equal(t, 0, ops["CertificateManager.ListCertificates"])
	assert.Equal(t, 0, ops["ListServerCertificates"])
}
