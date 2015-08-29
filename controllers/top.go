package controllers

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/cloudformation"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/cloudwatch"
	"github.com/convox/kernel/models"
)

func ClusterTop(rw http.ResponseWriter, r *http.Request) {
	name := aws.String(os.Getenv("RACK"))

	res, err := models.CloudFormation().DescribeStacks(&cloudformation.DescribeStacksInput{StackName: name})

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	stack := res.Stacks[0]

	outputs := make(map[string]string)

	for _, output := range stack.Outputs {
		outputs[*output.OutputKey] = *output.OutputValue
	}

	cluster := outputs["Cluster"]

	params := &cloudwatch.GetMetricStatisticsInput{
		MetricName: aws.String("CPUUtilization"),
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
		fmt.Println(err.Error())
		return
	}

	RenderJson(rw, resp)
}
