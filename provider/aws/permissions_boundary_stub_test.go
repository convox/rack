package aws_test

import (
	"path/filepath"
	"testing"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/pkg/test/awsutil"
	"github.com/stretchr/testify/require"
)

// cycleCreateStackBoundary answers a CreateStack call only when its request carries
// the boundary ARN, and answers with a distinct error so the caller can tell a
// matched request from an unmatched one.
func cycleCreateStackBoundary(pattern string) awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/",
			Body:       "/^Action=CreateStack&.*" + pattern + "/",
		},
		Response: awsutil.Response{
			StatusCode: 400,
			Body:       `<ErrorResponse><Error><Type>Sender</Type><Code>BoundaryRendered</Code><Message>boundary rendered</Message></Error></ErrorResponse>`,
		},
	}
}

const boundaryInTemplate = `PermissionsBoundary%22%3A\+%22arn%3Aaws%3Aiam%3A%3A123456789012%3Apolicy%2Fceiling%22`

func TestAppCreatePermissionsBoundary(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))

	for _, generation := range []string{"1", "2"} {
		provider := StubAwsProvider(
			cycleSystemDescribeStacksPermissionsBoundary,
			cycleCreateStackBoundary(boundaryInTemplate),
		)
		provider.Version = "test"

		_, err := provider.AppCreate("myapp", structs.AppCreateOptions{Generation: options.String(generation)})
		provider.Close()

		require.ErrorContains(t, err, "BoundaryRendered", "generation %s", generation)
	}
}

func TestSystemResourceCreatePermissionsBoundary(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))

	provider := StubAwsProvider(
		cycleResourceDescribeStacksMissing,
		cycleSystemDescribeStacksPermissionsBoundary,
		cycleCreateStackBoundary(`Parameters\.member\.\d+\.ParameterKey=PermissionsBoundary&Parameters\.member\.\d+\.ParameterValue=arn%3Aaws%3Aiam%3A%3A123456789012%3Apolicy%2Fceiling&`),
	)
	defer provider.Close()

	_, err := provider.SystemResourceCreate("s3", structs.ResourceCreateOptions{Name: options.String("mybucket")})

	require.ErrorContains(t, err, "BoundaryRendered")
}

var cycleResourceDescribeStacksMissing = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=DescribeStacks&StackName=convox-mybucket&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 400,
		Body:       `<ErrorResponse><Error><Type>Sender</Type><Code>ValidationError</Code><Message>Stack with id convox-mybucket does not exist</Message></Error></ErrorResponse>`,
	},
}
