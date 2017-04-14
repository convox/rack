package aws

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/api/structs"
)

func (p *AWSProvider) InstanceList() (structs.Instances, error) {
	ihash := map[string]structs.Instance{}

	req := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("tag:Rack"), Values: []*string{aws.String(os.Getenv("RACK"))}},
			{Name: aws.String("tag:aws:cloudformation:logical-id"), Values: []*string{aws.String("Instances")}},
		},
	}

	err := p.ec2().DescribeInstancesPages(req, func(res *ec2.DescribeInstancesOutput, last bool) bool {
		for _, r := range res.Reservations {
			for _, i := range r.Instances {
				ihash[cs(i.InstanceId, "")] = structs.Instance{
					Id:        cs(i.InstanceId, ""),
					PrivateIp: cs(i.PrivateIpAddress, ""),
					PublicIp:  cs(i.PublicIpAddress, ""),
					Status:    "",
					Started:   ct(i.LaunchTime),
				}
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	cis, err := p.describeContainerInstances()
	if err != nil {
		return nil, err
	}

	for _, cci := range cis.ContainerInstances {
		id := cs(cci.Ec2InstanceId, "")
		i := ihash[id]
		i.Agent = cb(cci.AgentConnected, false)
		i.Processes = int(ci(cci.RunningTasksCount, 0))
		ihash[id] = i
	}

	instances := structs.Instances{}

	for _, v := range ihash {
		instances = append(instances, v)
	}

	return instances, nil

	// ec2Res, err := p.ec2().DescribeInstances(&ec2.DescribeInstancesInput{})
	// if err != nil {
	//   return nil, err
	// }

	// ec2Instances := make(map[string]*ec2.Instance)
	// ec2Metrics := make(map[string]float64)

	// // collect instance data from EC2, and CPU Utilization from CloudWatch Metrics
	// for _, r := range ec2Res.Reservations {
	//   for _, i := range r.Instances {
	//     ec2Instances[*i.InstanceId] = i
	//     ec2Metrics[*i.InstanceId] = 0.0

	//     res, err := p.cloudwatch().GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
	//       Dimensions: []*cloudwatch.Dimension{
	//         {Name: aws.String("InstanceId"), Value: i.InstanceId},
	//       },
	//       EndTime:    aws.Time(time.Now()),
	//       MetricName: aws.String("CPUUtilization"),
	//       Namespace:  aws.String("AWS/EC2"),
	//       Period:     aws.Int64(5 * 60), // seconds
	//       StartTime:  aws.Time(time.Now().Add(time.Duration(-5) * time.Minute)),
	//       Statistics: []*string{aws.String("Average")},
	//     })
	//     if err != nil {
	//       continue
	//     }

	//     if len(res.Datapoints) > 0 {
	//       ec2Metrics[*i.InstanceId] = *res.Datapoints[0].Average / 100.0
	//     }
	//   }
	// }

	// var instances structs.Instances

	// // Calculate memory metrics from ECS DescribeContainerInstances
	// // We can not collect CPU metrics since we are not yet using ECS CPU reservations
	// for _, i := range ecsRes.ContainerInstances {
	//   var memory structs.InstanceResource

	//   for _, r := range i.RegisteredResources {
	//     switch *r.Name {
	//     case "MEMORY":
	//       memory.Total = int(*r.IntegerValue)
	//     }
	//   }

	//   for _, r := range i.RemainingResources {
	//     switch *r.Name {
	//     case "MEMORY":
	//       memory.Free = int(*r.IntegerValue)
	//       memory.Used = memory.Total - memory.Free
	//     }
	//   }

	//   // find the matching Instance from the EC2 response
	//   ec2Instance := ec2Instances[*i.Ec2InstanceId]

	//   // build up the struct
	//   instance := structs.Instance{
	//     Cpu:    ec2Metrics[*i.Ec2InstanceId],
	//     Memory: memory.PercentUsed(),
	//     Id:     *i.Ec2InstanceId,
	//   }

	//   if i.AgentConnected != nil {
	//     instance.Agent = *i.AgentConnected
	//   }

	//   if ec2Instance != nil {
	//     if ec2Instance.PrivateIpAddress != nil {
	//       instance.PrivateIp = *ec2Instance.PrivateIpAddress
	//     }

	//     if ec2Instance.PublicIpAddress != nil {
	//       instance.PublicIp = *ec2Instance.PublicIpAddress
	//     }

	//     if ec2Instance.LaunchTime != nil {
	//       instance.Started = *ec2Instance.LaunchTime
	//     }
	//   }

	//   if i.RunningTasksCount != nil {
	//     instance.Processes = int(*i.RunningTasksCount)
	//   }

	//   if i.Status != nil {
	//     instance.Status = strings.ToLower(*i.Status)
	//   }

	//   instances = append(instances, instance)
	// }

	// return instances, nil
}

func (p *AWSProvider) InstanceTerminate(id string) error {
	instances, err := p.InstanceList()
	if err != nil {
		return err
	}

	found := false

	for _, i := range instances {
		if i.Id == id {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("no such instance: %s", id)
	}

	_, err = p.autoscaling().TerminateInstanceInAutoScalingGroup(&autoscaling.TerminateInstanceInAutoScalingGroupInput{
		InstanceId:                     aws.String(id),
		ShouldDecrementDesiredCapacity: aws.Bool(false),
	})
	if err != nil {
		return err
	}

	return nil
}

// describeContainerInstances lists and describes all the ECS instances.
// It handles pagination for clusters > 100 instances.
func (p *AWSProvider) describeContainerInstances() (*ecs.DescribeContainerInstancesOutput, error) {
	instances := []*ecs.ContainerInstance{}
	var nextToken string

	for {
		res, err := p.listContainerInstances(&ecs.ListContainerInstancesInput{
			Cluster:   aws.String(p.Cluster),
			NextToken: &nextToken,
		})
		if ae, ok := err.(awserr.Error); ok && ae.Code() == "ClusterNotFoundException" {
			return nil, errorNotFound(fmt.Sprintf("cluster not found: %s", p.Cluster))
		}
		if err != nil {
			return nil, err
		}

		dres, err := p.ecs().DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
			Cluster:            aws.String(p.Cluster),
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
