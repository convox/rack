package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ecs"
)

var (
	AutoScaling    = autoscaling.New(session.New(), nil)
	CloudFormation = cloudformation.New(session.New(), nil)
	ECS            = ecs.New(session.New(), nil)
)

type Metrics struct {
	Cpu    int64
	Memory int64
	Width  int64
}

func Handler(ctx context.Context) error {
	largest, total, err := clusterMetrics()
	if err != nil {
		return err
	}

	fmt.Printf("largest = %+v\n", largest)
	fmt.Printf("total = %+v\n", total)

	desired, err := desiredCapacity(largest, total)
	if err != nil {
		return err
	}

	if err := autoscale(desired); err != nil {
		return err
	}

	return nil
}

func autoscale(desired int64) error {
	stack := os.Getenv("STACK")

	fmt.Printf("desired = %+v\n", desired)
	fmt.Printf("stack = %+v\n", stack)

	ds := fmt.Sprintf("%d", desired)

	res, err := CloudFormation.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stack),
	})
	if err != nil {
		return err
	}

	if len(res.Stacks) < 1 {
		return fmt.Errorf("could not find stack: %s", stack)
	}

	req := &cloudformation.UpdateStackInput{
		Capabilities:        []*string{aws.String("CAPABILITY_IAM")},
		Parameters:          []*cloudformation.Parameter{},
		StackName:           aws.String(stack),
		UsePreviousTemplate: aws.Bool(true),
	}

	for _, p := range res.Stacks[0].Parameters {
		switch *p.ParameterKey {
		case "InstanceCount":
			fmt.Printf("ds = %+v\n", ds)
			fmt.Printf("*p.ParameterValue = %+v\n", *p.ParameterValue)

			if ds == *p.ParameterValue {
				fmt.Println("no change")
				return nil
			}

			req.Parameters = append(req.Parameters, &cloudformation.Parameter{
				ParameterKey:   p.ParameterKey,
				ParameterValue: aws.String(ds),
			})
		default:
			req.Parameters = append(req.Parameters, &cloudformation.Parameter{
				ParameterKey:     p.ParameterKey,
				UsePreviousValue: aws.Bool(true),
			})
		}
	}

	if _, err := CloudFormation.UpdateStack(req); err != nil {
		return err
	}

	return nil
}

func clusterMetrics() (*Metrics, *Metrics, error) {
	// start with enough room for a single one-off run
	largest := &Metrics{Cpu: 128, Memory: 512}
	total := &Metrics{Cpu: 128, Memory: 512}

	req := &ecs.ListServicesInput{
		Cluster:    aws.String(os.Getenv("CLUSTER")),
		LaunchType: aws.String("EC2"),
	}

	for {
		res, err := ECS.ListServices(req)
		if err != nil {
			return nil, nil, err
		}

		fmt.Printf("ListServices page: %d\n", len(res.ServiceArns))

		dres, err := ECS.DescribeServices(&ecs.DescribeServicesInput{
			Cluster:  aws.String(os.Getenv("CLUSTER")),
			Services: res.ServiceArns,
		})
		if err != nil {
			return nil, nil, err
		}

		for _, s := range dres.Services {
			fmt.Printf("*s.ServiceName = %+v\n", *s.ServiceName)

			for _, d := range s.Deployments {
				res, err := ECS.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
					TaskDefinition: d.TaskDefinition,
				})
				if err != nil {
					return nil, nil, err
				}

				cpu := int64(0)
				mem := int64(0)
				width := int64(0)

				for _, cd := range res.TaskDefinition.ContainerDefinitions {
					cpu += ci(cd.Cpu)
					mem += max(ci(cd.Memory), ci(cd.MemoryReservation))

					if g := cd.DockerLabels["convox.generation"]; g == nil || *g != "2" {
						agent := false

						for _, pc := range s.PlacementConstraints {
							if pc.Type != nil && *pc.Type == "distinctInstance" {
								agent = true
								break
							}
						}

						fmt.Printf("agent = %+v\n", agent)
						fmt.Printf("len(cd.PortMappings) = %+v\n", len(cd.PortMappings))
						fmt.Printf("*s.DesiredCount = %+v\n", *s.DesiredCount)

						if len(cd.PortMappings) > 0 && !agent {
							width = *s.DesiredCount
						}
					}
				}

				if cpu > largest.Cpu {
					largest.Cpu = cpu
				}

				if mem > largest.Memory {
					largest.Memory = mem
				}

				if width > largest.Width {
					largest.Width = width
				}

				fmt.Printf("cpu = %+v\n", cpu)
				fmt.Printf("mem = %+v\n", mem)
				fmt.Printf("width = %+v\n", width)
				fmt.Printf("largest = %+v\n", largest)

				total.Cpu += (cpu * *s.DesiredCount)
				total.Memory += (mem * *s.DesiredCount)
			}
		}

		if res.NextToken == nil {
			break
		}

		req.NextToken = res.NextToken
	}

	return largest, total, nil
}

func desiredCapacity(largest, total *Metrics) (int64, error) {
	req := &ecs.ListContainerInstancesInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
		Status:  aws.String("ACTIVE"),
	}

	totalCount := int64(0)
	extraFit := int64(0)

	single := map[string]int64{}
	capacity := map[string]int64{}

	for {
		res, err := ECS.ListContainerInstances(req)
		if err != nil {
			return 0, err
		}

		fmt.Printf("ListContainerInstances page: %d\n", len(res.ContainerInstanceArns))

		totalCount += int64(len(res.ContainerInstanceArns))

		dres, err := ECS.DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
			Cluster:            aws.String(os.Getenv("CLUSTER")),
			ContainerInstances: res.ContainerInstanceArns,
		})
		if err != nil {
			return 0, err
		}

		for _, ci := range dres.ContainerInstances {
			remaining := map[string]int64{}

			for _, rr := range ci.RegisteredResources {
				if *rr.Type == "INTEGER" {
					single[*rr.Name] = *rr.IntegerValue
					capacity[*rr.Name] += *rr.IntegerValue
				}
			}

			for _, rr := range ci.RemainingResources {
				if *rr.Type == "INTEGER" {
					remaining[*rr.Name] = *rr.IntegerValue
				}
			}

			if remaining["CPU"] > largest.Cpu && remaining["MEMORY"] > largest.Memory {
				extraFit += 1
			}
		}

		fmt.Printf("capacity = %+v\n", capacity)
		fmt.Printf("single = %+v\n", single)

		if res.NextToken == nil {
			break
		}

		req.NextToken = res.NextToken
	}

	capcpu := int64(math.Floor((float64(capacity["CPU"]) - float64(total.Cpu)) / float64(single["CPU"])))
	capmem := int64(math.Floor((float64(capacity["MEMORY"]) - float64(total.Memory)) / float64(single["MEMORY"])))

	extraCapacity := min(capcpu, capmem)
	extraWidth := totalCount - largest.Width

	fmt.Printf("extraCapacity = %+v\n", extraCapacity)
	fmt.Printf("extraFit = %+v\n", extraFit)
	fmt.Printf("extraWidth = %+v\n", extraWidth)

	extra, err := strconv.ParseInt(os.Getenv("EXTRA"), 10, 64)
	if err != nil {
		return 0, err
	}

	desired := totalCount - (min(extraCapacity, extraFit, extraWidth) - extra)

	fmt.Printf("desired = %+v\n", desired)

	// minimum instance count is 3
	if desired < 3 {
		desired = 3
	}

	return desired, nil
}

func ci(ii ...*int64) int64 {
	for _, i := range ii {
		if i != nil && *i > 0 {
			return *i
		}
	}
	return 0
}

func max(ii ...int64) int64 {
	m := int64(0)
	for _, i := range ii {
		if i > m {
			m = i
		}
	}
	return m
}

func min(ii ...int64) int64 {
	m := int64(math.MaxInt64)
	for _, i := range ii {
		if i < m {
			m = i
		}
	}
	return m
}

func main() {
	lambda.Start(Handler)
}
