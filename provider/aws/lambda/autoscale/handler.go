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
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ecs"
)

const (
	HAInstanceCountParam   = "InstanceCount"
	NoHaInstanceCountParam = "NoHaInstanceCount"
)

var (
	CloudFormation     = cloudformation.New(session.New(), nil)
	ECS                = ecs.New(session.New(), nil)
	InstanceCountParam = HAInstanceCountParam
	IsHA               = os.Getenv("HIGH_AVAILABILITY") == "true"
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

	log("largest = %+v\n", largest)
	log("total = %+v\n", total)

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

	debug("desired = %+v\n", desired)
	debug("stack = %+v\n", stack)

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

	if !IsHA {
		InstanceCountParam = NoHaInstanceCountParam
	}

	for _, p := range res.Stacks[0].Parameters {
		switch *p.ParameterKey {
		case InstanceCountParam:
			debug("ds = %+v\n", ds)
			debug("*p.ParameterValue = %+v\n", *p.ParameterValue)

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

		debug("ListServices page: %d\n", len(res.ServiceArns))

		dres, err := ECS.DescribeServices(&ecs.DescribeServicesInput{
			Cluster:  aws.String(os.Getenv("CLUSTER")),
			Services: res.ServiceArns,
		})
		if err != nil {
			return nil, nil, err
		}

		for _, s := range dres.Services {
			debug("*s.ServiceName = %+v\n", *s.ServiceName)

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

						debug("agent = %+v\n", agent)
						debug("len(cd.PortMappings) = %+v\n", len(cd.PortMappings))
						debug("*s.DesiredCount = %+v\n", *s.DesiredCount)

						if len(cd.PortMappings) > 0 && !agent {
							width = *s.DesiredCount
						}
					}
				}

				if *s.DesiredCount > 0 {
					if cpu > largest.Cpu {
						largest.Cpu = cpu
					}

					if mem > largest.Memory {
						largest.Memory = mem
					}

					if width > largest.Width {
						largest.Width = width
					}
				}

				debug("cpu = %+v\n", cpu)
				debug("mem = %+v\n", mem)
				debug("width = %+v\n", width)
				debug("largest = %+v\n", largest)

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

	// the total number of instances
	totalCount := int64(0)

	// the number of instances that can fit the largest container
	extraFit := int64(0)

	// the attributes of a single instance
	single := map[string]int64{}

	// the total capacity of the cluster
	capacity := map[string]int64{}

	for {
		res, err := ECS.ListContainerInstances(req)
		if err != nil {
			return 0, err
		}

		debug("ListContainerInstances page: %d\n", len(res.ContainerInstanceArns))

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

			if remaining["CPU"] >= largest.Cpu && remaining["MEMORY"] >= largest.Memory {
				extraFit += 1
			}
		}

		debug("capacity = %+v\n", capacity)
		debug("single = %+v\n", single)

		if res.NextToken == nil {
			break
		}

		req.NextToken = res.NextToken
	}

	if largest.Memory > single["MEMORY"] || largest.Cpu > single["CPU"] {
		return 0, fmt.Errorf("largest container is bigger than a single instance")
	}

	// calculate the amount of extra capacity available in the cluster as a number of instances
	capcpu := int64(math.Floor((float64(capacity["CPU"]) - float64(total.Cpu)) / float64(single["CPU"])))
	capmem := int64(math.Floor((float64(capacity["MEMORY"]) - float64(total.Memory)) / float64(single["MEMORY"])))

	// the extra aggregate instance capacity is the smaller of extra cpu and extra memory values
	extraCapacity := min(capcpu, capmem)

	// the number of instances over the amount required to fit the widest service (gen1 loadbalancer-facing)
	extraWidth := totalCount - largest.Width

	debug("extraCapacity = %+v\n", extraCapacity)
	debug("extraFit = %+v\n", extraFit)
	debug("extraWidth = %+v\n", extraWidth)

	// comes from AutoscaleExtra
	extra, err := strconv.ParseInt(os.Getenv("EXTRA"), 10, 64)
	if err != nil {
		return 0, err
	}

	// total desired count is the current instance count minus the smallest calculated extra type plus the number of desired extra instances
	desired := totalCount - min(extraCapacity, extraFit, extraWidth) + extra

	debug("desired = %+v\n", desired)

	// minimum instance count is 3 for high available racks
	if IsHA && desired < 3 {
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

func debug(format string, a ...interface{}) {
	if os.Getenv("DEBUG") == "true" {
		fmt.Printf(format, a...)
	}
}

func log(format string, a ...interface{}) {
	fmt.Printf(format, a...)
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
