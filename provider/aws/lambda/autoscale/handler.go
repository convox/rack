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
	"github.com/aws/aws-sdk-go/service/ecs"
)

var (
	AutoScaling = autoscaling.New(session.New(), nil)
	ECS         = ecs.New(session.New(), nil)
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
	fmt.Printf("desired = %+v\n", desired)

	_, err := AutoScaling.SetDesiredCapacity(&autoscaling.SetDesiredCapacityInput{
		AutoScalingGroupName: aws.String(os.Getenv("ASG")),
		DesiredCapacity:      aws.Int64(desired),
	})
	if err != nil {
		return err
	}

	return nil
}

func clusterMetrics() (*Metrics, *Metrics, error) {
	// start with enough room for a single one-off run
	largest := &Metrics{Cpu: 128, Memory: 512}
	total := &Metrics{Cpu: 128, Memory: 512}

	res, err := ECS.ListServices(&ecs.ListServicesInput{
		Cluster:    aws.String(os.Getenv("CLUSTER")),
		LaunchType: aws.String("EC2"),
	})
	if err != nil {
		return nil, nil, err
	}

	for _, c := range chunk(res.ServiceArns, 10) {
		res, err := ECS.DescribeServices(&ecs.DescribeServicesInput{
			Cluster:  aws.String(os.Getenv("CLUSTER")),
			Services: c,
		})
		if err != nil {
			return nil, nil, err
		}

		for _, s := range res.Services {
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
						width = *s.DesiredCount
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

				total.Cpu += cpu
				total.Memory += mem
			}
		}
	}

	return largest, total, nil
}

func desiredCapacity(largest, total *Metrics) (int64, error) {
	res, err := ECS.ListContainerInstances(&ecs.ListContainerInstancesInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
		Status:  aws.String("ACTIVE"),
	})
	if err != nil {
		return 0, err
	}

	extraCapacity := int64(0)
	extraFit := int64(0)
	extraWidth := int64(len(res.ContainerInstanceArns)) - largest.Width

	single := map[string]int64{}
	capacity := map[string]int64{}
	num := int64(0)

	for _, c := range chunk(res.ContainerInstanceArns, 100) {
		res, err := ECS.DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
			Cluster:            aws.String(os.Getenv("CLUSTER")),
			ContainerInstances: c,
		})
		if err != nil {
			return 0, err
		}

		for _, ci := range res.ContainerInstances {
			remaining := map[string]int64{}
			num += 1

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
	}

	capcpu := int64(math.Floor((float64(capacity["CPU"]) - float64(total.Cpu)) / float64(single["CPU"])))
	capmem := int64(math.Floor((float64(capacity["MEMORY"]) - float64(total.Memory)) / float64(single["MEMORY"])))

	extraCapacity = min(capcpu, capmem)

	fmt.Printf("extraCapacity = %+v\n", extraCapacity)
	fmt.Printf("extraFit = %+v\n", extraFit)
	fmt.Printf("extraWidth = %+v\n", extraWidth)

	extra, err := strconv.ParseInt(os.Getenv("EXTRA"), 10, 64)
	if err != nil {
		return 0, err
	}

	return num - (min(extraCapacity, extraFit, extraWidth) - extra), nil
}

func chunk(ss []*string, size int) [][]*string {
	sss := [][]*string{}

	for {
		if len(ss) < size {
			return append(sss, ss)
		}

		sss = append(sss, ss[0:size])
		ss = ss[size:]
	}
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

// type Event struct {
//   Records []Record
// }

// type Message struct {
//   AutoScalingGroupName string
//   EC2InstanceID        string
//   LifecycleActionToken string
//   LifecycleHookName    string
//   LifecycleTransition  string
// }

// type Record struct {
//   Sns struct {
//     Message string
//   }
// }

// type Metadata struct {
//   Cluster string
//   Rack    string
// }

// func main() {
//   if len(os.Args) < 2 {
//     die(fmt.Errorf("must specify event as argument"))
//   }

//   data := []byte(os.Args[1])

//   var e Event

//   if err := json.Unmarshal(data, &e); err != nil {
//     die(err)
//   }

//   for _, r := range e.Records {
//     if err := handle(r); err != nil {
//       die(err)
//     }
//   }
// }

// func handle(r Record) error {
//   var m Message

//   if err := json.Unmarshal([]byte(r.Sns.Message), &m); err != nil {
//     return err
//   }

//   fmt.Printf("m = %+v\n", m)

//   if m.LifecycleTransition != "autoscaling:EC2_INSTANCE_TERMINATING" {
//     return nil
//   }

//   md, err := metadata()
//   if err != nil {
//     return err
//   }

//   fmt.Printf("md = %+v\n", md)

//   ci, err := containerInstance(md.Cluster, m.EC2InstanceID)
//   if err != nil {
//     return err
//   }

//   fmt.Printf("ci = %+v\n", ci)

//   cis, err := ECS.UpdateContainerInstancesState(&ecs.UpdateContainerInstancesStateInput{
//     ContainerInstances: []*string{
//       aws.String(ci),
//     },
//     Status:  aws.String("DRAINING"),
//     Cluster: aws.String(md.Cluster),
//   })
//   if err != nil {
//     return err
//   }

//   if len(cis.Failures) > 0 {
//     return fmt.Errorf("unable to drain instance: %s - %s", ci, *cis.Failures[0].Reason)
//   }

//   if err := waitForInstanceDrain(md.Cluster, ci); err != nil {
//     return err
//   }

//   fmt.Println("instance has been drained")

//   if _, err := ECS.DeregisterContainerInstance(&ecs.DeregisterContainerInstanceInput{
//     Cluster:           aws.String(md.Cluster),
//     ContainerInstance: aws.String(ci),
//     Force:             aws.Bool(true),
//   }); err != nil {
//     return err
//   }

//   _, err = AutoScaling.CompleteLifecycleAction(&autoscaling.CompleteLifecycleActionInput{
//     AutoScalingGroupName:  aws.String(m.AutoScalingGroupName),
//     InstanceId:            aws.String(m.EC2InstanceID),
//     LifecycleActionResult: aws.String("CONTINUE"),
//     LifecycleActionToken:  aws.String(m.LifecycleActionToken),
//     LifecycleHookName:     aws.String(m.LifecycleHookName),
//   })
//   if err != nil {
//     return err
//   }

//   fmt.Println("success")

//   return nil
// }

// func waitForInstanceDrain(cluster, ci string) error {

//   params := &ecs.ListTasksInput{
//     Cluster:           aws.String(cluster),
//     ContainerInstance: aws.String(ci),
//     DesiredStatus:     aws.String("RUNNING"),
//   }

//   tasks := []*string{}

//   for {
//     resp, err := ECS.ListTasks(params)
//     if err != nil {
//       return err
//     }

//     tasks = append(tasks, resp.TaskArns...)

//     if resp.NextToken == nil {
//       break
//     }

//     params.NextToken = resp.NextToken
//     time.Sleep(1 * time.Second)
//   }

//   if len(tasks) == 0 {
//     fmt.Println("no tasks to wait for")
//     return nil
//   }

//   input := &ecs.DescribeTasksInput{
//     Cluster: aws.String(cluster),
//     Tasks:   tasks,
//   }

//   if err := stopServicelessTasks(input); err != nil {
//     return err
//   }

//   fmt.Println("stopped service-less tasks")

//   return ECS.WaitUntilTasksStopped(input)
// }

// // stopServicelessTasks stops one-off tasks that do not belog to a ECS service.
// // For example, a scheduled task or running a process
// func stopServicelessTasks(input *ecs.DescribeTasksInput) error {

//   tasks, err := ECS.DescribeTasks(input)
//   if err != nil {
//     return err
//   }

//   for _, t := range tasks.Tasks {
//     // if the task isn't part of a service and wasn't started by ECS, stop it
//     if !strings.HasPrefix(*t.Group, "service:") && !strings.HasPrefix(*t.StartedBy, "ecs-svc") {
//       _, err := ECS.StopTask(&ecs.StopTaskInput{
//         Cluster: input.Cluster,
//         Reason:  aws.String("draining instance for termination"),
//         Task:    t.TaskArn,
//       })
//       if err != nil {
//         return err
//       }
//     }
//   }

//   return nil
// }

// func metadata() (*Metadata, error) {
//   var md Metadata

//   fres, err := Lambda.GetFunction(&lambda.GetFunctionInput{
//     FunctionName: aws.String(os.Getenv("AWS_LAMBDA_FUNCTION_NAME")),
//   })
//   if err != nil {
//     return nil, err
//   }

//   if err := json.Unmarshal([]byte(*fres.Configuration.Description), &md); err != nil {
//     return nil, err
//   }

//   return &md, nil
// }

// func containerInstance(cluster string, id string) (string, error) {
//   lreq := &ecs.ListContainerInstancesInput{
//     Cluster:    aws.String(cluster),
//     MaxResults: aws.Int64(10),
//   }

//   for {
//     lres, err := ECS.ListContainerInstances(lreq)
//     if err != nil {
//       return "", err
//     }

//     dres, err := ECS.DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
//       Cluster:            aws.String(cluster),
//       ContainerInstances: lres.ContainerInstanceArns,
//     })
//     if err != nil {
//       return "", err
//     }

//     for _, ci := range dres.ContainerInstances {
//       if *ci.Ec2InstanceId == id {
//         return *ci.ContainerInstanceArn, nil
//       }
//     }

//     if lres.NextToken == nil {
//       break
//     }

//     lreq.NextToken = lres.NextToken
//   }

//   return "", fmt.Errorf("could not find cluster instance: %s", id)
// }

// func die(err error) {
//   fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
//   os.Exit(1)
// }
