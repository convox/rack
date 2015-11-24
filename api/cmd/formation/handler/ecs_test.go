package handler

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/assert"
)

func TestECSServiceReplacementRequired(t *testing.T) {
	t.Skip("Need mock ECS DescribeServices to return existing load balancers")

	existing := ecs.LoadBalancer{
		LoadBalancerName: aws.String("convox"),
		ContainerName:    aws.String("web"),
		ContainerPort:    aws.Int64(80),
	}

	fmt.Printf("need to stub in %+v\n", existing)

	r := Request{}
	err := json.Unmarshal([]byte(`{"ResourceProperties": { "Cluster": "convox-Cluster-1JI343QBLSMYJ", "LoadBalancers": ["convox:web:80"] } }`), &r)

	required, err := ECSServiceReplacementRequired(r)
	assert.Nil(t, err)
	assert.False(t, required)

	// existing :: incoming => replacement required test cases
	// [] :: [] => false (no load balancer)
	// [] :: ["convox:web:80"] => true (new load balancer and port)
	// ["convox:web:80"] :: ["convox:web:80"] => false (no change)
	// ["convox:web:80"] :: ["convox:web:80", "convox:web:443"] => false (retains existing port 80)
	// ["convox:web:80"] :: ["convox:web:443"] => true (removed port 80)
	// ["convox:web:443"] :: ["convox:web:443"] => false (no change)
	// ["convox:web:443"] :: [] => true (removed all ports)
}
