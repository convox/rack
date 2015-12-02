package models

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/Godeps/_workspace/src/golang.org/x/crypto/ssh"
	"github.com/convox/rack/client"
)

var PrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEApv6WKDAIV9wacu32P8I/Y9xSbo3kLpQ5x1nuWlifoQnPc08zJ7K2Qppjl6Xa
xSqPEoP/SmcasRiALphiUcu2W9fEtG9G77awiYeaEgQa7UUuaqG+SnacKazLVyyh0Hp4cDiD9btm
b7PJ4d2Nu0l9GnjK/cvFvboLCOf1n0BEu4hhG29qogKetA0PDjyr8DyR4D2VfVoGjI/CybbDa8QU
Iesm5Q7ZLgjWgpSgeyS74JlwycC74YvBYOiT2b5kzYTntTYCd/gonhc17/YCAqv5B3DRBIsv+GRd
wuvdgqhLRvLjv4+EeNqsKGGTnSzu2dsDZyzPWbTj6OY1pMAFH4hjKQIDAQABAoIBAH/drhIFfU3w
7ZuU05nMXYdUGxYltVNpEbiwBo7NDyVagqrTOOMrttzWpG1ohFO2G6jcwywwOELj9Lo42gexiOdY
FnjmP5Wq+A/GcdVbqVaGQ11IjZEssrRCZ8xPE3OzYubij0AoBu5+5pT4dN60DYOofB3K2pVEj9B4
9BzFNBu2u1y8Pmz6PVqd+kGMtEGPIpuliCar7AMJx+ixMQr2JAuVKk84YzZu5Saza5o52vd8pBib
1vTZqSNU7kfUygaiCbVNopzVpMWhYYmIQhLAQwwXqV5A8sZuEHrlFQ/J9bv70CwxTbRcv8Cp+g3h
ty1cBvlQeferND9ahOmYsUyi5YECgYEA1/K4/aJXOP0g/p9XLBPmQ4uce5TTFOyH0Ceob7539piz
YTepo7jTwcU1V6sy1ABwKepk9dhc7iN1qq4XdqxAWbHxnuMqY7Ci+f2LVxBHuC+ze+/l3aEy/8nI
/BSnJofg3lI7jFx8qg28Mv4OkoVYJr5NDirDCdpXmo6hZxLMm9kCgYEAxfeH3k14N5EaT2tLhVk9
texk4j1F/iT8/W45UFBy0Mjsuqvdzoz28kvW/Dg9A7+oAZJuzKYnZwlco7bgrNfu1Pp7BZAO3Jjm
hJFUmG7vAXwoiehOh3Sw0wCnLcj1pvfnXdI/ywuq5y0UR/TUo1u+fffnmW52ibheK6XfYmlA/9EC
gYBY5o/Jut91kp/WsvpMJxUQkZUmOyp63rU6uFjbR+pTFqIiT6wCvsBOcUV4hf4y0MtcNibCHwSC
9Q4n6eu260rCokL6SkLVL46oo/yNJyKfbOPTDfvvtcEtFIEtZcM6VY35eJkTO7AGwgjMZVLSdxrH
OGi4gFoy4DRYaIeBy3d4YQKBgA4H9kxOR1gA49F/NFIWOiZ7w8a5Ow3BR2Ea/9ruaMTdiNHOPqFW
ImaX83va7JAodFrwKwQ8PoyyACvmWVRG1bmoqzGAvVzrRWNd/ZX0PuJnD2R+35oALkw2PqMjHC4i
YfanYTgd8pYB/u7+rleJuB2rhXG9f49RTvNfBU8vUJkRAoGAHkJmSZ0v1hV54eTQz3oVCEq7XRE4
Uy8Ltxh2MaN9W4uX5mQT/Yzm7IR8s4GEUheawEniKUznANlLLsyI5ur1/DAgN8dAVwzNGixWtjhY
LimQaEcEOKSBA6hgbEZQeDwnLFLXZgZL0ajlzfVy4oypLpafOaHW5KbNurSXlMxYL58=
-----END RSA PRIVATE KEY-----`

type Instance client.Instance

type InstanceResource struct {
	Total int `json:"total"`
	Free  int `json:"free"`
	Used  int `json:"used"`
}

func (ir InstanceResource) PercentUsed() float64 {
	return float64(ir.Used) / float64(ir.Total)
}

func InstanceSSH(id, command string, height, width int, rw io.ReadWriter) error {
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

	signer, err := ssh.ParsePrivateKey([]byte(PrivateKey))
	if err != nil {
		return err
	}
	config := &ssh.ClientConfig{
		User: "ec2-user",
		Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
	}
	conn, err := ssh.Dial("tcp", *instance.PublicIpAddress+":22", config)
	if err != nil {
		return err
	}
	defer conn.Close()
	session, err := conn.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = rw
	session.Stdin = rw
	session.Stderr = rw

	fmt.Println("running", command)

	if command != "" {
		err = session.Run(command)
		if err != nil {
			return err
		}
	} else {
		modes := ssh.TerminalModes{
			ssh.ECHOCTL:       0,
			ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
			ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		}
		// Request pseudo terminal
		if err := session.RequestPty("xterm", width, height, modes); err != nil {
			return err
		}
		// Start remote shell
		if err := session.Shell(); err != nil {
			return err
		}

		session.Wait()
	}

	return nil
}

func (s *System) GetInstances() ([]*Instance, error) {
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
			Agent:     *i.AgentConnected,
			Cpu:       truncate(cpu.PercentUsed(), 4),
			Memory:    truncate(memory.PercentUsed(), 4),
			Id:        *i.Ec2InstanceId,
			Ip:        *ec2Instance.PublicIpAddress,
			Processes: int(*i.RunningTasksCount),
			Status:    strings.ToLower(*i.Status),
		}

		instances = append(instances, instance)
	}

	return instances, nil
}
