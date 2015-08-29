package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/cloudwatch"
	"github.com/convox/kernel/models"
)

func ClusterTop(rw http.ResponseWriter, r *http.Request) {
	params := &cloudwatch.GetMetricStatisticsInput{
		MetricName: aws.String("CPUUtilization"),
		StartTime:  aws.Time(time.Now().Add(-2 * time.Minute)),
		EndTime:    aws.Time(time.Now()),
		Period:     aws.Long(60),
		Namespace:  aws.String("AWS/ECS"),
		Statistics: []*string{ // Required
			aws.String("Maximum"),
			aws.String("Average"),
		},
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("ClusterName"),
				Value: aws.String("convo-Clust-1VY3S3P89LZUN"),
			},
		},
	}

	resp, err := models.CloudWatch().GetMetricStatistics(params)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	enc, err := json.Marshal(resp)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println(string(enc))
}
