package aws

import (
	"testing"

	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/require"
)

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
