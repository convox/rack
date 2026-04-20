package aws

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/require"
)

// renderServiceTemplate renders formation/service.json.tmpl for the named
// service. The stp key set matches the production construction at
// provider/aws/releases.go:372-384.
//
// Note: this duplicates a subset of formationTemplate() logic (provider/aws/
// template.go:157) but uses a relative path rooted at the test CWD
// (provider/aws/) rather than the production relative-to-repo-root path.
func renderServiceTemplate(t *testing.T, m *manifest.Manifest, serviceName string) []byte {
	t.Helper()

	var svc manifest.Service
	found := false
	for _, s := range m.Services {
		if s.Name == serviceName {
			svc = s
			found = true
			break
		}
	}
	require.True(t, found, "service %q not in manifest", serviceName)

	stp := map[string]interface{}{
		"App":            "testapp",
		"Autoscale":      false,
		"Build":          &structs.Build{Id: "B12345", App: "testapp"},
		"DeploymentMin":  50,
		"DeploymentMax":  200,
		"Manifest":       m,
		"Password":       "testpass",
		"Release":        &structs.Release{Id: "R12345", App: "testapp", Build: "B12345"},
		"Service":        svc,
		"Tags":           svc.Tags,
		"WildcardDomain": false,
	}

	path := filepath.Join("formation", "service.json.tmpl")
	tpl, err := template.New("service.json.tmpl").Funcs(formationHelpers()).ParseFiles(path)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, tpl.Execute(&buf, stp))

	// Round-trip through json.Unmarshal/MarshalIndent to mirror formationTemplate's
	// pretty-printed normalization.
	var v interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &v))
	out, err := json.MarshalIndent(v, "", "  ")
	require.NoError(t, err)
	return out
}

// extractResource parses rendered CF JSON and returns the named resource's
// body as a raw json.RawMessage for targeted assertions.
func extractResource(t *testing.T, rendered []byte, logicalID string) json.RawMessage {
	t.Helper()
	var parsed struct {
		Resources map[string]json.RawMessage `json:"Resources"`
	}
	require.NoError(t, json.Unmarshal(rendered, &parsed))
	res, ok := parsed.Resources[logicalID]
	require.True(t, ok, "resource %q not found in rendered template", logicalID)
	return res
}

// loadManifestFromTestdata loads pkg/manifest/testdata/<name>.yml. provider/aws
// cannot reuse the pkg/manifest_test private helpers.
func loadManifestFromTestdata(t *testing.T, name string) (*manifest.Manifest, error) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "pkg", "manifest", "testdata", name+".yml"))
	if err != nil {
		return nil, err
	}
	return manifest.Load(data, nil)
}

func TestManifestToServiceNlbPorts(t *testing.T) {
	tests := []struct {
		name string
		in   []manifest.ServiceNLBPort
		want []structs.ServiceNlbPort
	}{
		{
			name: "nil input returns nil",
			in:   nil,
			want: nil,
		},
		{
			name: "empty non-nil input returns nil",
			in:   []manifest.ServiceNLBPort{},
			want: nil,
		},
		{
			name: "one public port",
			in: []manifest.ServiceNLBPort{
				{Port: 8443, Protocol: "tcp", ContainerPort: 3000, Scheme: "public"},
			},
			want: []structs.ServiceNlbPort{
				{Port: 8443, Protocol: "tcp", ContainerPort: 3000, Scheme: "public"},
			},
		},
		{
			name: "one internal port",
			in: []manifest.ServiceNLBPort{
				{Port: 9443, Protocol: "tcp", ContainerPort: 8080, Scheme: "internal"},
			},
			want: []structs.ServiceNlbPort{
				{Port: 9443, Protocol: "tcp", ContainerPort: 8080, Scheme: "internal"},
			},
		},
		{
			name: "port equals container port",
			in: []manifest.ServiceNLBPort{
				{Port: 8443, Protocol: "tcp", ContainerPort: 8443, Scheme: "public"},
			},
			want: []structs.ServiceNlbPort{
				{Port: 8443, Protocol: "tcp", ContainerPort: 8443, Scheme: "public"},
			},
		},
		{
			name: "mixed schemes public first preserves order",
			in: []manifest.ServiceNLBPort{
				{Port: 8443, Protocol: "tcp", ContainerPort: 8443, Scheme: "public"},
				{Port: 9443, Protocol: "tcp", ContainerPort: 8080, Scheme: "internal"},
			},
			want: []structs.ServiceNlbPort{
				{Port: 8443, Protocol: "tcp", ContainerPort: 8443, Scheme: "public"},
				{Port: 9443, Protocol: "tcp", ContainerPort: 8080, Scheme: "internal"},
			},
		},
		{
			name: "mixed schemes internal first preserves order",
			in: []manifest.ServiceNLBPort{
				{Port: 9443, Protocol: "tcp", ContainerPort: 8080, Scheme: "internal"},
				{Port: 8443, Protocol: "tcp", ContainerPort: 8443, Scheme: "public"},
			},
			want: []structs.ServiceNlbPort{
				{Port: 9443, Protocol: "tcp", ContainerPort: 8080, Scheme: "internal"},
				{Port: 8443, Protocol: "tcp", ContainerPort: 8443, Scheme: "public"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manifestToServiceNlbPorts(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}

// nlbTCPOnlyManifest is inline so the regression check doesn't depend on
// external fixture files that might be incidentally touched by this PR.
var nlbTCPOnlyManifest = []byte(`services:
  api:
    image: example/api
    nlb:
      - port: 443
        protocol: tcp
        containerPort: 8080
        scheme: public
`)

func TestServiceTemplateNLBTCPListenerRegression(t *testing.T) {
	m, err := manifest.Load(nlbTCPOnlyManifest, nil)
	require.NoError(t, err)
	rendered := renderServiceTemplate(t, m, "api")
	listener := extractResource(t, rendered, "NLBListener443")

	var parsed struct {
		Properties map[string]json.RawMessage
	}
	require.NoError(t, json.Unmarshal(listener, &parsed))

	_, hasSsl := parsed.Properties["SslPolicy"]
	require.False(t, hasSsl, "TCP NLB listener must not have SslPolicy property")
	_, hasCerts := parsed.Properties["Certificates"]
	require.False(t, hasCerts, "TCP NLB listener must not have Certificates property")

	var proto string
	require.NoError(t, json.Unmarshal(parsed.Properties["Protocol"], &proto))
	require.Equal(t, "TCP", proto)
}

func TestServiceTemplateNLBTLSListenerMinimal(t *testing.T) {
	m, err := loadManifestFromTestdata(t, "nlb-tls")
	require.NoError(t, err)
	rendered := renderServiceTemplate(t, m, "api")
	listener := extractResource(t, rendered, "NLBListener443")

	var parsed struct {
		Properties struct {
			Protocol     string
			SslPolicy    string
			Certificates []struct {
				CertificateArn string
			}
		}
	}
	require.NoError(t, json.Unmarshal(listener, &parsed))
	require.Equal(t, "TLS", parsed.Properties.Protocol)
	require.Equal(t, "ELBSecurityPolicy-TLS13-1-2-2021-06", parsed.Properties.SslPolicy)
	require.Len(t, parsed.Properties.Certificates, 1)
	require.Equal(t,
		"arn:aws:acm:us-east-1:123456789012:certificate/00000000-0000-0000-0000-000000000001",
		parsed.Properties.Certificates[0].CertificateArn,
	)
}

func TestServiceTemplateNLBTLSListenerInternal(t *testing.T) {
	m, err := loadManifestFromTestdata(t, "nlb-tls-internal")
	require.NoError(t, err)
	rendered := renderServiceTemplate(t, m, "api")
	listener := extractResource(t, rendered, "NLBListener443")
	require.Contains(t, string(listener), "NLBInternalArn",
		"internal-scheme listener must import NLBInternalArn")
}

func TestServiceTemplateNLBMixedTCPAndTLS(t *testing.T) {
	m, err := loadManifestFromTestdata(t, "nlb-tls-mixed")
	require.NoError(t, err)
	rendered := renderServiceTemplate(t, m, "api")

	tls := extractResource(t, rendered, "NLBListener443")
	require.True(t, strings.Contains(string(tls), `"TLS"`))
	require.True(t, strings.Contains(string(tls), "ELBSecurityPolicy-TLS13-1-2-2021-06"))
	require.True(t, strings.Contains(string(tls), "CertificateArn"))

	tcp := extractResource(t, rendered, "NLBListener8080")
	require.True(t, strings.Contains(string(tcp), `"TCP"`))
	require.False(t, strings.Contains(string(tcp), "SslPolicy"))
	require.False(t, strings.Contains(string(tcp), "Certificates"))
}

func TestServiceTemplateNLBTLSListenerIAMCert(t *testing.T) {
	m, err := loadManifestFromTestdata(t, "nlb-tls-iam")
	require.NoError(t, err)
	rendered := renderServiceTemplate(t, m, "api")
	listener := extractResource(t, rendered, "NLBListener443")
	require.True(t, strings.Contains(string(listener),
		"arn:aws:iam::123456789012:server-certificate/my-server-cert"))
}
