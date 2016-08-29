package models

import (
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

type Process struct {
	Id      string    `json:"id"`
	App     string    `json:"app"`
	Command string    `json:"command"`
	Host    string    `json:"host"`
	Image   string    `json:"image"`
	Name    string    `json:"name"`
	Ports   []string  `json:"ports"`
	Release string    `json:"release"`
	Cpu     float64   `json:"cpu"`
	Memory  float64   `json:"memory"`
	Started time.Time `json:"started"`

	binds       []string `json:"-"`
	containerId string   `json:"-"`
	taskArn     string   `json:"-"`
}

type Processes []*Process

// DescribeContainerInstances lists and describes all the ECS instances.
// It handles pagination for clusters > 100 instances.
func DescribeContainerInstances() (*ecs.DescribeContainerInstancesOutput, error) {
	instances := []*ecs.ContainerInstance{}
	var nextToken string

	for {
		res, err := ECS().ListContainerInstances(&ecs.ListContainerInstancesInput{
			Cluster:   aws.String(os.Getenv("CLUSTER")),
			NextToken: &nextToken,
		})
		if err != nil {
			return nil, err
		}

		dres, err := ECS().DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
			Cluster:            aws.String(os.Getenv("CLUSTER")),
			ContainerInstances: res.ContainerInstanceArns,
		})
		if err != nil {
			return nil, err
		}

		instances = append(instances, dres.ContainerInstances...)

		// No more container results
		if res.NextToken == nil {
			break
		}

		// set the nextToken to be used for the next iteration
		nextToken = *res.NextToken
	}

	return &ecs.DescribeContainerInstancesOutput{
		ContainerInstances: instances,
	}, nil
}
