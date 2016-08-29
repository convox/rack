package aws

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/api/structs"
)

func (p *AWSProvider) ProcessAttach(pid string, stream io.ReadWriter) error {
	arn, err := p.taskArnFromPid(pid)
	if err != nil {
		return err
	}

	timeout := time.After(10 * time.Second)
	tick := time.Tick(1 * time.Second)

	// wait until the task is no longer PENDING
	for {
		select {
		case <-tick:
			task, err := p.describeTask(arn)
			if err != nil {
				return err
			}
			if *task.LastStatus != "PENDING" {
				break
			}
		case <-timeout:
			return fmt.Errorf("timeout starting process")
		}
	}

	return p.taskStream(arn, stream)
}

func (p *AWSProvider) ProcessExec(app, pid, command string, stream io.ReadWriter) error {
	return nil
}

// ProcessList returns a list of processes for an app
func (p *AWSProvider) ProcessList(app string) (structs.Processes, error) {
	rres, err := p.describeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(fmt.Sprintf("%s-%s", p.Rack, app)),
	})
	if err != nil {
		return nil, err
	}

	services := []string{}

	for _, r := range rres.StackResources {
		switch *r.ResourceType {
		case "AWS::ECS::Service", "Custom::ECSService":
			services = append(services, *r.PhysicalResourceId)
		}
	}

	tasks := []*string{}

	for _, s := range services {
		tres, err := p.ecs().ListTasks(&ecs.ListTasksInput{
			Cluster:     aws.String(p.Cluster),
			ServiceName: aws.String(s),
		})
		if err != nil {
			return nil, err
		}
		for _, arn := range tres.TaskArns {
			tasks = append(tasks, arn)
		}
	}

	tres, err := p.ecs().DescribeTasks(&ecs.DescribeTasksInput{
		Cluster: aws.String(p.Cluster),
		Tasks:   tasks,
	})

	pss := structs.Processes{}

	for _, task := range tres.Tasks {
		if len(task.Containers) != 1 {
			return nil, fmt.Errorf("invalid task: %s", *task.TaskDefinitionArn)
		}

		cd, err := p.containerDefinitionForTask(*task.TaskDefinitionArn)
		if err != nil {
			return nil, err
		}

		ci, err := p.containerInstance(*task.ContainerInstanceArn)
		if err != nil {
			return nil, err
		}

		host, err := p.describeInstance(*ci.Ec2InstanceId)
		if err != nil {
			return nil, err
		}

		env := map[string]string{}
		for _, e := range cd.Environment {
			env[*e.Name] = *e.Value
		}

		container := task.Containers[0]
		parts := strings.Split(*container.ContainerArn, "-")

		ports := []string{}
		for _, p := range container.NetworkBindings {
			ports = append(ports, fmt.Sprintf("%d:%d", *p.HostPort, *p.ContainerPort))
		}

		cmd := make([]string, len(cd.Command))
		for i, c := range cd.Command {
			if strings.Contains(*c, " ") {
				cmd[i] = fmt.Sprintf("%q", *c)
			} else {
				cmd[i] = *c
			}
		}

		ps := structs.Process{
			ID:       parts[len(parts)-1],
			App:      app,
			Name:     *container.Name,
			Release:  env["RELEASE"],
			Command:  strings.Join(cmd, " "),
			Host:     *host.PrivateIpAddress,
			Image:    *cd.Image,
			Instance: *ci.Ec2InstanceId,
			Ports:    ports,
			Started:  *task.StartedAt,
		}

		// guard for nil
		if task.StartedAt != nil {
			ps.Started = *task.StartedAt
		}

		pss = append(pss, ps)
	}

	return pss, nil
}

func (p *AWSProvider) ProcessRun(app, process string, opts structs.ProcessRunOptions) (*structs.Process, error) {
	task := fmt.Sprintf("%s-%s-%s", os.Getenv("RACK"), app, process)

	if opts.Release != "" {
		t, err := p.taskDefinitionForRelease(app, process, opts.Release)
		if err != nil {
			return nil, err
		}
		task = t
	}

	fmt.Printf("opts = %+v\n", opts)
	fmt.Printf("task = %+v\n", task)

	req := &ecs.RunTaskInput{
		Cluster:        aws.String(os.Getenv("CLUSTER")),
		Count:          aws.Int64(1),
		StartedBy:      aws.String("convox"),
		TaskDefinition: aws.String(task),
	}

	if opts.Command != "" {
		req.Overrides = &ecs.TaskOverride{
			ContainerOverrides: []*ecs.ContainerOverride{
				&ecs.ContainerOverride{
					Name: aws.String(process),
					Command: []*string{
						aws.String("sh"),
						aws.String("-c"),
						aws.String(opts.Command),
					},
				},
			},
		}
	}

	res, err := p.ecs().RunTask(req)
	if err != nil {
		return nil, err
	}
	if len(res.Tasks) != 1 || len(res.Tasks[0].Containers) != 1 {
		if len(res.Failures) > 0 {
			switch *res.Failures[0].Reason {
			case "RESOURCE:MEMORY":
				return nil, fmt.Errorf("not enough memory available to start process")
			}
		}
		return nil, fmt.Errorf("could not start process")
	}

	proc := &structs.Process{
		ID: arnToPid(*res.Tasks[0].TaskArn),
	}

	return proc, nil
}

func (p *AWSProvider) ProcessStop(app, pid string) error {
	return nil
}

func arnToPid(arn string) string {
	parts := strings.Split(arn, "-")
	return parts[len(parts)-1]
}

func (p *AWSProvider) containerDefinitionForTask(arn string) (*ecs.ContainerDefinition, error) {
	res, err := p.ecs().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(arn),
	})
	if err != nil {
		return nil, err
	}
	if len(res.TaskDefinition.ContainerDefinitions) != 1 {
		return nil, fmt.Errorf("invalid container definitions for task: %s", arn)
	}

	return res.TaskDefinition.ContainerDefinitions[0], nil
}

func (p *AWSProvider) containerInstance(id string) (*ecs.ContainerInstance, error) {
	res, err := p.ecs().DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(p.Cluster),
		ContainerInstances: []*string{aws.String(id)},
	})
	if err != nil {
		return nil, err
	}
	if len(res.ContainerInstances) != 1 {
		return nil, fmt.Errorf("could not find container instance: %s", id)
	}

	return res.ContainerInstances[0], nil
}

func (p *AWSProvider) describeInstance(id string) (*ec2.Instance, error) {
	res, err := p.ec2().DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(id)},
	})
	fmt.Printf("res = %+v\n", res)
	fmt.Printf("err = %+v\n", err)
	if err != nil {
		return nil, err
	}
	if len(res.Reservations) != 1 || len(res.Reservations[0].Instances) != 1 {
		return nil, fmt.Errorf("could not find instance: %s", id)
	}

	return res.Reservations[0].Instances[0], nil
}

func (p *AWSProvider) describeTask(arn string) (*ecs.Task, error) {
	res, err := p.ecs().DescribeTasks(&ecs.DescribeTasksInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
		Tasks:   []*string{aws.String(arn)},
	})
	if err != nil {
		return nil, err
	}
	if len(res.Tasks) != 1 {
		return nil, fmt.Errorf("could not fetch process status")
	}
	return res.Tasks[0], nil
}

func (p *AWSProvider) taskDefinitionForRelease(app, process, id string) (string, error) {
	prefix := fmt.Sprintf("%s-%s-%s", os.Getenv("RACK"), app, process)

	for {
		res, err := p.ecs().ListTaskDefinitions(&ecs.ListTaskDefinitionsInput{
			FamilyPrefix: aws.String(prefix),
		})

		fmt.Printf("res = %+v\n", res)
		fmt.Printf("err = %+v\n", err)

		break
	}

	return "", nil

}

func (p *AWSProvider) taskArnFromPid(pid string) (string, error) {
	token := ""

	for {
		req := &ecs.ListTasksInput{
			Cluster: aws.String(os.Getenv("CLUSTER")),
		}
		if token != "" {
			req.NextToken = aws.String(token)
		}

		res, err := p.ecs().ListTasks(req)
		fmt.Printf("res = %+v\n", res)
		fmt.Printf("err = %+v\n", err)

		for _, arn := range res.TaskArns {
			if arnToPid(*arn) == pid {
				return *arn, nil
			}
		}

		if res.NextToken == nil {
			break
		}

		token = *res.NextToken
	}

	return "", fmt.Errorf("could not find process")
}

func (p *AWSProvider) taskStream(arn string, stream io.ReadWriter) error {
	res, err := p.ecs().DescribeTasks(&ecs.DescribeTasksInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
		Tasks:   []*string{aws.String(arn)},
	})
	if err != nil {
		return err
	}
	fmt.Printf("res = %+v\n", res)
	fmt.Printf("err = %+v\n", err)
	return nil
}

// task, err := ECS().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
//   TaskDefinition: aws.String(taskDefinitionArn),
// })
// if err != nil {
//   return err
// }

// td, _, err := findAppDefinitions(process, releaseID, *task.TaskDefinition.Family, 20)
// if err != nil {
//   return err

// } else if td != nil {
//   taskDefinitionArn = *td.TaskDefinitionArn

// } else {
//   // If reached, the release exist but doesn't have a task definition (isn't promoted).
//   // Create a task definition to run that release.

//   var cd *ecs.ContainerDefinition
//   for _, cd = range task.TaskDefinition.ContainerDefinitions {
//     if *cd.Name == process {
//       break
//     }
//     cd = nil
//   }
//   if cd == nil {
//     return fmt.Errorf("unable to find container for process %s and release %s", process, releaseID)
//   }

//   env := structs.Environment{}
//   env.LoadRaw(release.Env)

//   for _, containerKV := range cd.Environment {
//     for key, value := range env {

//       if *containerKV.Name == "RELEASE" {
//         *containerKV.Value = releaseID
//         break

//       }

//       if *containerKV.Name == key {
//         *containerKV.Value = value
//         break
//       }
//     }
//   }

//   taskInput := &ecs.RegisterTaskDefinitionInput{
//     ContainerDefinitions: []*ecs.ContainerDefinition{
//       cd,
//     },
//     Family:  task.TaskDefinition.Family,
//     Volumes: []*ecs.Volume{},
//   }

//   resp, err := ECS().RegisterTaskDefinition(taskInput)
//   if err != nil {
//     return err
//   }

//   taskDefinitionArn = *resp.TaskDefinition.TaskDefinitionArn
// }

// return "", nil
