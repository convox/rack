package aws

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/pkg/test/awsutil"
)

func TestMigratedAmiPath(t *testing.T) {
	tests := []struct {
		name    string
		param   string
		current string
		want    string
		wantOk  bool
	}{
		{
			name:    "DefaultAmi on retired al2 path",
			param:   "DefaultAmi",
			current: "/aws/service/ecs/optimized-ami/amazon-linux-2/recommended/image_id",
			want:    "/aws/service/ecs/optimized-ami/amazon-linux-2023/recommended/image_id",
			wantOk:  true,
		},
		{
			name:    "DefaultAmiArm on retired al2 path",
			param:   "DefaultAmiArm",
			current: "/aws/service/ecs/optimized-ami/amazon-linux-2/arm64/recommended/image_id",
			want:    "/aws/service/ecs/optimized-ami/amazon-linux-2023/arm64/recommended/image_id",
			wantOk:  true,
		},
		{
			name:    "DefaultAmi with custom path",
			param:   "DefaultAmi",
			current: "/custom/ami/path",
			wantOk:  false,
		},
		{
			name:    "DefaultAmi already migrated",
			param:   "DefaultAmi",
			current: "/aws/service/ecs/optimized-ami/amazon-linux-2023/recommended/image_id",
			wantOk:  false,
		},
		{
			name:    "DefaultAmi with arm path does not cross match",
			param:   "DefaultAmi",
			current: "/aws/service/ecs/optimized-ami/amazon-linux-2/arm64/recommended/image_id",
			wantOk:  false,
		},
		{
			name:    "unrelated parameter",
			param:   "InstanceType",
			current: "/aws/service/ecs/optimized-ami/amazon-linux-2/recommended/image_id",
			wantOk:  false,
		},
		{
			name:    "DefaultAmi empty value",
			param:   "DefaultAmi",
			current: "",
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := migratedAmiPath(tt.param, tt.current)
			if ok != tt.wantOk || got != tt.want {
				t.Fatalf("migratedAmiPath(%q, %q) = (%q, %v), want (%q, %v)", tt.param, tt.current, got, ok, tt.want, tt.wantOk)
			}
		})
	}
}

func stubRackDescribeStacks(t *testing.T, params string) *Provider {
	handler := awsutil.NewHandler([]awsutil.Cycle{
		{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DescribeStacks&StackName=convox&Version=2010-05-15`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: fmt.Sprintf(`<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
					<DescribeStacksResult>
						<Stacks>
							<member>
								<StackId>arn:aws:cloudformation:us-east-1:778743527532:stack/convox/eb743e00-7d8e-11e5-8280-50ba0727c06e</StackId>
								<StackName>convox</StackName>
								<StackStatus>UPDATE_COMPLETE</StackStatus>
								<CreationTime>2015-10-28T16:14:09.590Z</CreationTime>
								<Parameters>%s</Parameters>
							</member>
						</Stacks>
					</DescribeStacksResult>
				</DescribeStacksResponse>`, params),
			},
		},
	})

	s := httptest.NewServer(handler)
	t.Cleanup(s.Close)

	t.Setenv("AWS_ACCESS_KEY_ID", "test-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

	return &Provider{
		Region:    "us-test-1",
		Endpoint:  s.URL,
		Rack:      "convox",
		SkipCache: true,
	}
}

func TestRackHasRetiredAmi(t *testing.T) {
	tests := []struct {
		name   string
		params string
		want   bool
	}{
		{
			name: "retired al2 path present",
			params: `<member><ParameterKey>DefaultAmi</ParameterKey><ParameterValue>/aws/service/ecs/optimized-ami/amazon-linux-2/recommended/image_id</ParameterValue></member>
				<member><ParameterKey>DefaultAmiArm</ParameterKey><ParameterValue>/aws/service/ecs/optimized-ami/amazon-linux-2023/arm64/recommended/image_id</ParameterValue></member>`,
			want: true,
		},
		{
			name: "already migrated",
			params: `<member><ParameterKey>DefaultAmi</ParameterKey><ParameterValue>/aws/service/ecs/optimized-ami/amazon-linux-2023/recommended/image_id</ParameterValue></member>
				<member><ParameterKey>DefaultAmiArm</ParameterKey><ParameterValue>/aws/service/ecs/optimized-ami/amazon-linux-2023/arm64/recommended/image_id</ParameterValue></member>`,
			want: false,
		},
		{
			name:   "no ami parameters",
			params: `<member><ParameterKey>InstanceType</ParameterKey><ParameterValue>t2.small</ParameterValue></member>`,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := stubRackDescribeStacks(t, tt.params)
			got, err := p.rackHasRetiredAmi()
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("rackHasRetiredAmi() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRackHasRetiredAmiDescribeError(t *testing.T) {
	handler := awsutil.NewHandler([]awsutil.Cycle{})
	s := httptest.NewServer(handler)
	t.Cleanup(s.Close)

	t.Setenv("AWS_ACCESS_KEY_ID", "test-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

	p := &Provider{Region: "us-test-1", Endpoint: s.URL, Rack: "convox", SkipCache: true}

	if _, err := p.rackHasRetiredAmi(); err == nil {
		t.Fatal("rackHasRetiredAmi() should return an error when the stack cannot be described")
	}
}

func TestIsNoUpdatesError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "cloudformation no updates validation error",
			err:  awserr.New("ValidationError", "No updates are to be performed.", nil),
			want: true,
		},
		{
			name: "other validation error",
			err:  awserr.New("ValidationError", "Parameter validation failed", nil),
			want: false,
		},
		{
			name: "no updates message with different code",
			err:  awserr.New("Throttling", "No updates are to be performed.", nil),
			want: false,
		},
		{
			name: "plain error",
			err:  errors.New("No updates are to be performed."),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNoUpdatesError(tt.err); got != tt.want {
				t.Fatalf("isNoUpdatesError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestFormationAmiDefaultsMatchMigrationTargets(t *testing.T) {
	data, err := os.ReadFile("formation/rack.json")
	if err != nil {
		t.Fatal(err)
	}

	var formation struct {
		Parameters map[string]struct {
			Default interface{} `json:"Default"`
		} `json:"Parameters"`
	}

	if err := json.Unmarshal(data, &formation); err != nil {
		t.Fatal(err)
	}

	for param, m := range retiredAmiPaths {
		fp, ok := formation.Parameters[param]
		if !ok {
			t.Fatalf("parameter %s not found in formation/rack.json", param)
		}

		if fp.Default != m.new {
			t.Errorf("parameter %s default is %q, want %q", param, fp.Default, m.new)
		}
	}
}
