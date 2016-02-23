package models

import (
	"fmt"
	"io"
	"math/rand"
	"os"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/rack/Godeps/_workspace/src/golang.org/x/crypto/ssh"
	"github.com/convox/rack/api/provider"
)

var (
	StatusCodePrefix = "F1E49A85-0AD7-4AEF-A618-C249C6E6568D:" // needs to be random
)

func InstanceKeyroll() error {
	keyname := fmt.Sprintf("%s-keypair-%d", os.Getenv("RACK"), (rand.Intn(8999) + 1000))
	keypair, err := EC2().CreateKeyPair(&ec2.CreateKeyPairInput{
		KeyName: &keyname,
	})

	if err != nil {
		return err
	}

	env, err := provider.SettingsGet(os.Getenv("RACK"))

	if err != nil {
		return err
	}

	env["InstancePEM"] = *keypair.KeyMaterial

	err = provider.SettingsSet(os.Getenv("RACK"), env)

	if err != nil {
		return err
	}

	app, err := provider.AppGet(os.Getenv("RACK"))

	if err != nil {
		return err
	}

	err = AppUpdateParams(app, map[string]string{
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

	env, err := provider.SettingsGet(os.Getenv("RACK"))

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

func exitCode(err error) int {
	if ee, ok := err.(*ssh.ExitError); ok {
		return ee.Waitmsg.ExitStatus()
	}

	if err != nil {
		return -1
	}

	return 0
}
