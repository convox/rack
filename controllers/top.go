package controllers

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/models"
)

func ClusterTop(rw http.ResponseWriter, r *http.Request) {
	name := aws.String(os.Getenv("RACK"))

	res, err := models.CloudFormation().DescribeStacks(&cloudformation.DescribeStacksInput{StackName: name})

	if err != nil {
		RenderError(rw, err)
		return
	}

	if len(res.Stacks) == 0 {
		RenderError(rw, fmt.Errorf("Stack %s does not exist", os.Getenv("RACK")))
		return
	}

	stack := res.Stacks[0]

	outputs := make(map[string]string)

	for _, output := range stack.Outputs {
		outputs[*output.OutputKey] = *output.OutputValue
	}

	cluster := outputs["Cluster"]

	params := &cloudwatch.GetMetricStatisticsInput{
		MetricName: aws.String(mux.Vars(r)["metric"]),
		StartTime:  aws.Time(time.Now().Add(-2 * time.Minute)),
		EndTime:    aws.Time(time.Now()),
		Period:     aws.Long(60),
		Namespace:  aws.String("AWS/ECS"),
		Statistics: []*string{ // Required
			aws.String("Maximum"),
			aws.String("Average"),
			aws.String("Minimum"),
		},
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("ClusterName"),
				Value: aws.String(cluster),
			},
		},
	}

	resp, err := models.CloudWatch().GetMetricStatistics(params)

	if err != nil {
		RenderError(rw, err)
		return
	}

	RenderJson(rw, resp)
}
