package aws

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/rack/structs"
	"golang.org/x/crypto/ssh"
)

func (p *AWSProvider) InstanceKeyroll() error {
	key := fmt.Sprintf("%s-keypair-%d", os.Getenv("RACK"), (rand.Intn(8999) + 1000))

	res, err := p.ec2().CreateKeyPair(&ec2.CreateKeyPairInput{
		KeyName: aws.String(key),
	})
	if err != nil {
		return err
	}

	if err := p.SettingPut("instance-key", *res.KeyMaterial); err != nil {
		return err
	}

	if err := p.updateStack(p.Rack, "", map[string]string{"Key": key}); err != nil {
		return err
	}

	return nil
}

func (p *AWSProvider) InstanceList() (structs.Instances, error) {
	ihash := map[string]structs.Instance{}

	req := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("tag:Rack"), Values: []*string{aws.String(os.Getenv("RACK"))}},
			{Name: aws.String("tag:aws:cloudformation:logical-id"), Values: []*string{aws.String("Instances"), aws.String("SpotInstances")}},
			{Name: aws.String("instance-state-name"), Values: []*string{aws.String("pending"), aws.String("running"), aws.String("shutting-down"), aws.String("stopping")}},
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

	cis, err := p.listAndDescribeContainerInstances()
	if err != nil {
		return nil, err
	}

	for _, cci := range cis.ContainerInstances {
		id := cs(cci.Ec2InstanceId, "")
		i := ihash[id]

		i.Agent = cb(cci.AgentConnected, false)
		i.Processes = int(ci(cci.RunningTasksCount, 0))
		i.Status = strings.ToLower(cs(cci.Status, "unknown"))

		var cpu, memory instanceResource

		for _, r := range cci.RegisteredResources {
			switch *r.Name {
			case "CPU":
				cpu.Total = int(ci(r.IntegerValue, 0))
			case "MEMORY":
				memory.Total = int(ci(r.IntegerValue, 0))
			}
		}

		for _, r := range cci.RemainingResources {
			switch *r.Name {
			case "CPU":
				cpu.Free = int(ci(r.IntegerValue, 0))
				cpu.Used = cpu.Total - cpu.Free
			case "MEMORY":
				memory.Free = int(ci(r.IntegerValue, 0))
				memory.Used = memory.Total - memory.Free
			}
		}

		i.Cpu = cpu.PercentUsed()
		i.Memory = memory.PercentUsed()

		ihash[id] = i
	}

	instances := structs.Instances{}

	for _, v := range ihash {
		instances = append(instances, v)
	}

	sort.Sort(instances)

	return instances, nil
}

func (p *AWSProvider) InstanceShell(id string, rw io.ReadWriter, opts structs.InstanceShellOptions) error {
	res, err := p.ec2().DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("instance-id"), Values: []*string{aws.String(id)}},
		},
		MaxResults: aws.Int64(1000),
	})
	if err != nil {
		return err
	}
	if len(res.Reservations) < 1 {
		return fmt.Errorf("instance not found")
	}

	instance := res.Reservations[0].Instances[0]

	key, err := p.SettingGet("instance-key")
	if err != nil {
		return fmt.Errorf("no instance key found")
	}

	// configure SSH client
	signer, err := ssh.ParsePrivateKey([]byte(key))
	if err != nil {
		return err
	}

	config := &ssh.ClientConfig{
		User:            "ec2-user",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	ip := *instance.PrivateIpAddress
	if os.Getenv("DEVELOPMENT") == "true" {
		ip = *instance.PublicIpAddress
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", ip), config)
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

	if opts.Terminal != nil {
		width := 0
		height := 0

		if opts.Width != nil {
			width = *opts.Width
		}

		if opts.Height != nil {
			height = *opts.Height
		}

		modes := ssh.TerminalModes{
			ssh.ECHOCTL:       0,
			ssh.TTY_OP_ISPEED: 56000, // input speed = 56kbaud
			ssh.TTY_OP_OSPEED: 56000, // output speed = 56kbaud
		}
		if err := session.RequestPty(*opts.Terminal, width, height, modes); err != nil {
			return err
		}
	}

	code := 0

	if opts.Command != nil {
		if err := session.Start(*opts.Command); err != nil {
			return err
		}
	} else {
		if err := session.Shell(); err != nil {
			return err
		}
	}

	if err := session.Wait(); err != nil {
		if ee, ok := err.(*ssh.ExitError); ok {
			code = ee.Waitmsg.ExitStatus()
		}
	}

	if _, err := rw.Write([]byte(fmt.Sprintf("%s%d\n", StatusCodePrefix, code))); err != nil {
		return err
	}

	return nil
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

type instanceResource struct {
	Total int `json:"total"`
	Free  int `json:"free"`
	Used  int `json:"used"`
}

func (ir instanceResource) PercentUsed() float64 {
	return float64(ir.Used) / float64(ir.Total)
}
