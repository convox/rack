package workers

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/api/models"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
)

type Instance struct {
	Id string

	ASG    bool
	Check  bool
	Docker bool
	ECS    bool

	Unhealthy bool
}

type Instances map[string]Instance

func StartCluster() {
	var log = logger.New("ns=cluster_monitor")

	defer recoverWith(func(err error) {
		helpers.Error(log, err)
	})

	// Report cluster size one time on start
	system, err := models.GetSystem()

	if err != nil {
		log.Error(err)
	}

	helpers.TrackEvent("kernel-cluster-monitor", fmt.Sprintf("count=%d type=%s", system.Count, system.Type))

	for _ = range time.Tick(5 * time.Minute) {
		log.Log("tick")

		instances := Instances{}

		err := instances.describeASG()

		if err != nil {
			log.Error(err)
			continue
		}

		err = instances.describeECS()

		if err != nil {
			log.Error(err)
			continue
		}

		// TODO: Add an instances.testDocker() call to the mission critical path

		// Test if ASG Instance is registered and connected in ECS cluster
		for _, i := range instances {
			if !i.ASG {
				// TODO: Rogue instance?! Terminate?
				continue
			}

			if !i.ECS {
				// Not registered or not connected => set Unhealthy
				_, err := models.AutoScaling().SetInstanceHealth(
					&autoscaling.SetInstanceHealthInput{
						HealthStatus:             aws.String("Unhealthy"),
						InstanceId:               aws.String(i.Id),
						ShouldRespectGracePeriod: aws.Bool(true),
					},
				)

				i.Unhealthy = true

				if err != nil {
					log.Error(err)
					continue
				}
			}
		}

		log.Log(instances.log())
	}
}

func (instances Instances) describeASG() error {
	resources, err := models.ListResources(os.Getenv("RACK"))

	res, err := models.AutoScaling().DescribeAutoScalingGroups(
		&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{
				aws.String(resources["Instances"].Id),
			},
		},
	)

	if err != nil {
		return err
	}

	for _, i := range res.AutoScalingGroups[0].Instances {
		instance := instances[*i.InstanceId]

		instance.Id = *i.InstanceId
		instance.ASG = *i.LifecycleState == "InService"

		instances[*i.InstanceId] = instance
	}

	return nil
}

func (instances Instances) describeECS() error {
	res, err := models.ECS().ListContainerInstances(
		&ecs.ListContainerInstancesInput{
			Cluster: aws.String(os.Getenv("CLUSTER")),
		},
	)

	if err != nil {
		return err
	}

	dres, err := models.ECS().DescribeContainerInstances(
		&ecs.DescribeContainerInstancesInput{
			Cluster:            aws.String(os.Getenv("CLUSTER")),
			ContainerInstances: res.ContainerInstanceArns,
		},
	)

	if err != nil {
		return err
	}

	for _, i := range dres.ContainerInstances {
		instance := instances[*i.Ec2InstanceId]

		instance.Id = *i.Ec2InstanceId
		instance.ECS = *i.AgentConnected

		instances[*i.Ec2InstanceId] = instance
	}

	return nil
}

func (instances Instances) log() string {
	var asgIds, ecsIds, unhealthyIds []string

	for _, i := range instances {
		if i.ASG {
			asgIds = append(asgIds, i.Id)
		}

		if i.ECS {
			ecsIds = append(ecsIds, i.Id)
		}

		if i.Unhealthy {
			unhealthyIds = append(unhealthyIds, i.Id)
		}
	}

	sort.Strings(asgIds)
	sort.Strings(ecsIds)
	sort.Strings(unhealthyIds)

	return fmt.Sprintf("count=%v connected='%v' healthy='%v' marked='%s'",
		len(instances),
		strings.Join(ecsIds, ","),
		strings.Join(asgIds, ","),
		strings.Join(unhealthyIds, ","),
	)
}

func (instances Instances) testDocker() error {
	for _, i := range instances {
		instance := instances[i.Id]

		res, err := models.EC2().DescribeInstances(&ec2.DescribeInstancesInput{
			Filters: []*ec2.Filter{
				&ec2.Filter{Name: aws.String("instance-id"), Values: []*string{&i.Id}},
			},
		})

		if err != nil {
			return err
		}

		if len(res.Reservations) != 1 || len(res.Reservations[0].Instances) != 1 {
			return fmt.Errorf("could not describe container instance")
		}

		ip := *res.Reservations[0].Instances[0].PrivateIpAddress

		if os.Getenv("DEVELOPMENT") == "true" {
			ip = *res.Reservations[0].Instances[0].PublicIpAddress
		}

		d, err := docker.NewClient(fmt.Sprintf("http://%s:2376", ip))

		if err != nil {
			return err
		}

		err = d.PullImage(docker.PullImageOptions{
			Repository: "busybox",
		}, docker.AuthConfiguration{})

		if err != nil {
			return err
		}

		instance.Docker = true

		dres, err := d.CreateContainer(docker.CreateContainerOptions{
			Config: &docker.Config{
				Cmd:   []string{"sh", "-c", `dmesg | grep "Remounting filesystem read-only"`},
				Image: "busybox",
			},
		})

		if err != nil {
			return err
		}

		err = d.StartContainer(dres.ID, nil)

		if err != nil {
			return err
		}

		code, err := d.WaitContainer(dres.ID)

		if err != nil {
			return err
		}

		// grep exit status is 0 if any line was selected, 1 otherwise
		// no "Remounting" selected is healthy
		if code == 1 {
			instance.Check = true
		}

		instances[i.Id] = instance
	}

	return nil
}
