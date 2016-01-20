package models

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
	"github.com/convox/rack/Godeps/_workspace/src/golang.org/x/crypto/ssh"
)

type Instance struct {
	Agent     bool      `json:"agent"`
	Cpu       float64   `json:"cpu"`
	Id        string    `json:"id"`
	Ip        string    `json:"ip"`
	Memory    float64   `json:"memory"`
	Processes int       `json:"processes"`
	Status    string    `json:"status"`
	Started   time.Time `json:"started"`
}

type InstanceResource struct {
	Total int `json:"total"`
	Free  int `json:"free"`
	Used  int `json:"used"`
}

func (ir InstanceResource) PercentUsed() float64 {
	return float64(ir.Used) / float64(ir.Total)
}

func InstanceKeyroll() error {
	keyname := fmt.Sprintf("%s-keypair-%d", os.Getenv("RACK"), (rand.Intn(8999) + 1000))
	keypair, err := EC2().CreateKeyPair(&ec2.CreateKeyPairInput{
		KeyName: &keyname,
	})

	if err != nil {
		return err
	}

	env, err := GetRackSettings()

	if err != nil {
		return err
	}

	env["InstancePEM"] = *keypair.KeyMaterial
	err = PutRackSettings(env)

	if err != nil {
		return err
	}

	app, err := GetApp(os.Getenv("RACK"))
	if err != nil {
		return err
	}

	err = app.UpdateParams(map[string]string{
		"Key": keyname,
	})
	if err != nil {
		return err
	}

	return nil
}

func InstanceSSH(id, command, term string, height, width int, rw io.ReadWriter) error {
	instanceIds := []*string{&id}
	ec2Res, err := EC2().DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{Name: aws.String("instance-id"), Values: instanceIds},
		},
	})

	if err != nil {
		return err
	}

	instance := ec2Res.Reservations[0].Instances[0]

	env, err := GetRackSettings()
	if err != nil {
		return err
	}

	// configure SSH client
	signer, err := ssh.ParsePrivateKey([]byte(env["InstancePEM"]))
	if err != nil {
		return err
	}
	config := &ssh.ClientConfig{
		User: "ec2-user",
		Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
	}

	ipAddress := *instance.PrivateIpAddress
	if os.Getenv("DEVELOPMENT") == "true" {
		ipAddress = *instance.PublicIpAddress
	}

	conn, err := ssh.Dial("tcp", ipAddress+":22", config)
	if err != nil {
		return err
	}
	defer conn.Close()
	session, err := conn.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Setup I/O
	session.Stdout = rw
	session.Stdin = rw
	session.Stderr = rw

	// Setup terminal if requested
	if term != "" {
		modes := ssh.TerminalModes{
			ssh.ECHOCTL:       0,
			ssh.TTY_OP_ISPEED: 56000, // input speed = 56kbaud
			ssh.TTY_OP_OSPEED: 56000, // output speed = 56kbaud
		}
		// Request pseudo terminal
		if err := session.RequestPty(term, width, height, modes); err != nil {
			return err
		}
	}

	code := 0
	// Start remote shell
	if command != "" {
		if err := session.Start(command); err != nil {
			return err
		}
	} else {
		if err := session.Shell(); err != nil {
			return err
		}
	}

	err = session.Wait()

	if err != nil {
		code = exitCode(err)
	}

	_, err = rw.Write([]byte(fmt.Sprintf("%s%d\n", StatusCodePrefix, code)))

	if err != nil {
		return err
	}

	return nil
}

func ListInstances() ([]*Instance, error) {
	res, err := ECS().ListContainerInstances(
		&ecs.ListContainerInstancesInput{
			Cluster: aws.String(os.Getenv("CLUSTER")),
		},
	)

	if err != nil {
		return nil, err
	}

	ecsRes, err := ECS().DescribeContainerInstances(
		&ecs.DescribeContainerInstancesInput{
			Cluster:            aws.String(os.Getenv("CLUSTER")),
			ContainerInstances: res.ContainerInstanceArns,
		},
	)

	if err != nil {
		return nil, err
	}

	var instanceIds []*string
	for _, i := range ecsRes.ContainerInstances {
		instanceIds = append(instanceIds, i.Ec2InstanceId)
	}

	ec2Res, err := EC2().DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{Name: aws.String("instance-id"), Values: instanceIds},
		},
	})

	if err != nil {
		return nil, err
	}

	ec2Instances := make(map[string]*ec2.Instance)
	for _, r := range ec2Res.Reservations {
		for _, i := range r.Instances {
			ec2Instances[*i.InstanceId] = i
		}
	}

	var instances []*Instance
	for _, i := range ecsRes.ContainerInstances {
		// figure out the CPU and memory metrics
		var cpu, memory InstanceResource

		for _, r := range i.RegisteredResources {
			switch *r.Name {
			case "CPU":
				cpu.Total = int(*r.IntegerValue)
			case "MEMORY":
				memory.Total = int(*r.IntegerValue)
			}
		}

		for _, r := range i.RemainingResources {
			switch *r.Name {
			case "CPU":
				cpu.Free = int(*r.IntegerValue)
				cpu.Used = cpu.Total - cpu.Free
			case "MEMORY":
				memory.Free = int(*r.IntegerValue)
				memory.Used = memory.Total - memory.Free
			}
		}

		// find the matching Instance from the EC2 response
		ec2Instance := ec2Instances[*i.Ec2InstanceId]

		// build up the struct
		instance := &Instance{
			Cpu:    truncate(cpu.PercentUsed(), 4),
			Memory: truncate(memory.PercentUsed(), 4),
			Id:     *i.Ec2InstanceId,
		}

		if i.AgentConnected != nil {
			instance.Agent = *i.AgentConnected
		}

		if ec2Instance != nil {
			if ec2Instance.PublicIpAddress != nil {
				instance.Ip = *ec2Instance.PublicIpAddress
			}

			if ec2Instance.LaunchTime != nil {
				instance.Started = *ec2Instance.LaunchTime
			}
		}

		if i.RunningTasksCount != nil {
			instance.Processes = int(*i.RunningTasksCount)
		}

		if i.Status != nil {
			instance.Status = strings.ToLower(*i.Status)
		}

		instances = append(instances, instance)
	}

	return instances, nil
}

func exitCode(err error) int {
	if ee, ok := err.(*ssh.ExitError); ok {
		return ee.Waitmsg.ExitStatus()
	}

	if err != nil {
		return -1
	}

	return 0
}

func (i *Instance) Docker() (*docker.Client, error) {
	return Docker(fmt.Sprintf("http://%s:2376", i.Ip))
}
