package handler

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/api/crypt"
	"github.com/convox/rack/api/models"
)

func HandleECSService(req Request) (string, map[string]string, error) {
	switch req.RequestType {
	case "Create":
		fmt.Println("CREATING SERVICE")
		fmt.Printf("req %+v\n", req)
		return ECSServiceCreate(req)
	case "Update":
		fmt.Println("UPDATING SERVICE")
		fmt.Printf("req %+v\n", req)
		return ECSServiceUpdate(req)
	case "Delete":
		fmt.Println("DELETING SERVICE")
		fmt.Printf("req %+v\n", req)
		return ECSServiceDelete(req)
	}

	return "invalid", nil, fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

func HandleECSTaskDefinition(req Request) (string, map[string]string, error) {
	switch req.RequestType {
	case "Create":
		fmt.Println("CREATING TASK")
		fmt.Printf("req %+v\n", req)
		return ECSTaskDefinitionCreate(req)
	case "Update":
		fmt.Println("UPDATING TASK")
		fmt.Printf("req %+v\n", req)
		return ECSTaskDefinitionCreate(req)
	case "Delete":
		fmt.Println("DELETING TASK")
		fmt.Printf("req %+v\n", req)
		return ECSTaskDefinitionDelete(req)
	}

	return "invalid", nil, fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

func ECSServiceCreate(req Request) (string, map[string]string, error) {
	count, err := strconv.Atoi(req.ResourceProperties["DesiredCount"].(string))
	if err != nil {
		return "invalid", nil, err
	}

	r := &ecs.CreateServiceInput{
		Cluster:        aws.String(req.ResourceProperties["Cluster"].(string)),
		DesiredCount:   aws.Int64(int64(count)),
		ServiceName:    aws.String(req.ResourceProperties["Name"].(string) + "-" + generateId("S", 10)),
		TaskDefinition: aws.String(req.ResourceProperties["TaskDefinition"].(string)),
	}

	balancers := req.ResourceProperties["LoadBalancers"].([]interface{})

	if len(balancers) > 0 {
		r.Role = aws.String(req.ResourceProperties["Role"].(string))
	}

	for _, balancer := range balancers {
		parts := strings.Split(balancer.(string), "||")

		if len(parts) != 3 {
			return "invalid", nil, fmt.Errorf("invalid load balancer specification: %s", balancer.(string))
		}

		name := parts[0]
		ps := parts[1]
		port, _ := strconv.Atoi(parts[2])

		lb := &ecs.LoadBalancer{
			ContainerName: aws.String(ps),
			ContainerPort: aws.Int64(int64(port)),
		}

		if strings.HasPrefix(name, "arn:") {
			lb.TargetGroupArn = aws.String(name)
		} else {
			lb.LoadBalancerName = aws.String(name)
		}

		r.LoadBalancers = append(r.LoadBalancers, lb)

		// Despite the ECS Create Service API docs, you can only specify a single load balancer name and port. Specifying more than one results in
		// Failed to update resource. InvalidParameterException: load balancers can have at most 1 items. status code: 400, request id: 0839710e-9227-11e5-8a2f-015e938a7aea
		// https://github.com/aws/aws-cli/issues/1362
		// Therefore break after adding the first load balancer mapping to the CreateServiceInput
		break
	}

	if req.ResourceProperties["DeploymentMinimumPercent"] != nil && req.ResourceProperties["DeploymentMaximumPercent"] != nil {
		min, err := strconv.Atoi(req.ResourceProperties["DeploymentMinimumPercent"].(string))

		if err != nil {
			return "could not parse DeploymentMinimumPercent", nil, err
		}

		max, err := strconv.Atoi(req.ResourceProperties["DeploymentMaximumPercent"].(string))

		if err != nil {
			return "could not parse DeploymentMaximumPercent", nil, err
		}

		r.DeploymentConfiguration = &ecs.DeploymentConfiguration{
			MinimumHealthyPercent: aws.Int64(int64(min)),
			MaximumPercent:        aws.Int64(int64(max)),
		}
	}

	res, err := ECS(req).CreateService(r)

	if err != nil {
		return "invalid", nil, err
	}

	return *res.Service.ServiceArn, nil, nil
}

func ECSServiceUpdate(req Request) (string, map[string]string, error) {
	count, err := strconv.Atoi(req.ResourceProperties["DesiredCount"].(string))
	if err != nil {
		return "invalid", nil, err
	}

	// arn:aws:ecs:us-east-1:922560784203:service/sinatra-SZXTRXEMYEY
	parts := strings.Split(req.PhysicalResourceId, "/")
	name := parts[1]

	replace, err := ECSServiceReplacementRequired(req)

	if err != nil {
		return "invalid", nil, err
	}

	if replace {
		return ECSServiceCreate(req)
	}

	r := &ecs.UpdateServiceInput{
		Cluster:        aws.String(req.ResourceProperties["Cluster"].(string)),
		Service:        aws.String(name),
		DesiredCount:   aws.Int64(int64(count)),
		TaskDefinition: aws.String(req.ResourceProperties["TaskDefinition"].(string)),
	}

	if req.ResourceProperties["DeploymentMinimumPercent"] != nil && req.ResourceProperties["DeploymentMaximumPercent"] != nil {
		min, err := strconv.Atoi(req.ResourceProperties["DeploymentMinimumPercent"].(string))

		if err != nil {
			return "could not parse DeploymentMinimumPercent", nil, err
		}

		max, err := strconv.Atoi(req.ResourceProperties["DeploymentMaximumPercent"].(string))

		if err != nil {
			return "could not parse DeploymentMaximumPercent", nil, err
		}

		r.DeploymentConfiguration = &ecs.DeploymentConfiguration{
			MinimumHealthyPercent: aws.Int64(int64(min)),
			MaximumPercent:        aws.Int64(int64(max)),
		}
	}

	res, err := ECS(req).UpdateService(r)

	if err != nil {
		return req.PhysicalResourceId, nil, err
	}

	return *res.Service.ServiceArn, nil, nil
}

// According to the ECS Docs (http://docs.aws.amazon.com/AmazonECS/latest/developerguide/update-service.html):
// To change the load balancer name, the container name, or the container port associated with a service load balancer configuration, you must create a new service.
func ECSServiceReplacementRequired(req Request) (bool, error) {
	incoming := []string{}
	existing := make(map[string]bool)

	res, err := ECS(req).DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(req.ResourceProperties["Cluster"].(string)),
		Services: []*string{aws.String(req.PhysicalResourceId)},
	})

	if err != nil {
		return false, err
	}

	balancers := req.ResourceProperties["LoadBalancers"].([]interface{})

	for _, ilb := range balancers {
		incoming = append(incoming, ilb.(string))
	}

	if len(balancers) > 0 {
		if req.ResourceProperties["Role"].(string) != *res.Services[0].RoleArn {
			return true, nil
		}
	}

	// NOTE: Despite the Service APIs taking and returning a list, at most one balancer:container:port mapping will be set
	for _, lb := range res.Services[0].LoadBalancers {
		existing[fmt.Sprintf("%s:%s:%d", *lb.LoadBalancerName, *lb.ContainerName, *lb.ContainerPort)] = true
	}

	// update retains no load balancers
	if len(incoming) == 0 && len(existing) == 0 {
		return false, nil
	}

	// update retains one existing service port mapping
	for _, lb := range incoming {
		if existing[lb] {
			return false, nil
		}
	}

	// update creates or removes existing service port mapping
	return true, nil
}

func ECSServiceDelete(req Request) (string, map[string]string, error) {
	cluster := req.ResourceProperties["Cluster"].(string)

	// arn:aws:ecs:us-east-1:922560784203:service/sinatra-SZXTRXEMYEY
	parts := strings.Split(req.PhysicalResourceId, "/")
	name := parts[1]

	// Set Desired to 0
	_, err := ECS(req).UpdateService(&ecs.UpdateServiceInput{
		Cluster:      aws.String(cluster),
		Service:      aws.String(name),
		DesiredCount: aws.Int64(0),
	})

	// go ahead and mark the delete good if the service is not found
	if ae, ok := err.(awserr.Error); ok {
		if ae.Code() == "ServiceNotFoundException" {
			return req.PhysicalResourceId, nil, nil
		}
	}

	// signal DELETE_FAILED to cloudformation
	if err != nil {
		fmt.Printf("ECS UpdateService error: %s\n", err)
		return req.PhysicalResourceId, nil, err
	}

	// Help move Desired to 0 by stopping all tasks
	tasks, err := ECS(req).ListTasks(&ecs.ListTasksInput{
		Cluster:     aws.String(cluster),
		ServiceName: aws.String(name),
	})

	if err != nil {
		fmt.Printf("ECS ListTasks error: %s\n", err)
	} else {
		for _, arn := range tasks.TaskArns {
			_, err = ECS(req).StopTask(&ecs.StopTaskInput{
				Cluster: aws.String(cluster),
				Task:    arn,
			})

			if err != nil {
				fmt.Printf("ECS StopTask error: %s\n", err)
			}
		}
	}

	// Delete service, sleeping/retrying for 2 minutes if the error is:
	// Failed to delete resource. InvalidParameterException: The service cannot be stopped while deployments are active.
	var derr error

	for i := 0; i < 12; i++ {
		_, derr = ECS(req).DeleteService(&ecs.DeleteServiceInput{
			Cluster: aws.String(cluster),
			Service: aws.String(name),
		})

		// sleep and retry
		if ae, ok := derr.(awserr.Error); ok {
			if ae.Code() == "InvalidParameterException" {
				time.Sleep(10 * time.Second)
				continue
			}
		}

		// signal DELETE_FAILED to cloudformation
		if derr != nil {
			fmt.Printf("error: %s\n", derr)
			return req.PhysicalResourceId, nil, derr
		}

		// signal DELETE_COMPLETE to cloudformation
		return req.PhysicalResourceId, nil, nil
	}

	// signal DELETE_FAILED to cloudformation
	return req.PhysicalResourceId, nil, derr
}

func ECSTaskDefinitionCreate(req Request) (string, map[string]string, error) {
	// return "", fmt.Errorf("fail")

	tasks := req.ResourceProperties["Tasks"].([]interface{})

	r := &ecs.RegisterTaskDefinitionInput{
		Family: aws.String(req.ResourceProperties["Name"].(string)),
	}

	// get environment from S3 URL
	// 'Environment' is a CloudFormation Template Property that references 'Environment' CF Parameter with S3 URL
	// S3 body may be encrypted with KMS key
	var env models.Environment

	if envUrl, ok := req.ResourceProperties["Environment"].(string); ok && envUrl != "" {
		res, err := http.Get(envUrl)

		if err != nil {
			return "invalid", nil, err
		}

		defer res.Body.Close()

		data, err := ioutil.ReadAll(res.Body)

		if key, ok := req.ResourceProperties["Key"].(string); ok && key != "" {
			cr := crypt.New(*Region(&req), os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"))
			cr.AwsToken = os.Getenv("AWS_SESSION_TOKEN")

			dec, err := cr.Decrypt(key, data)

			if err != nil {
				return "invalid", nil, err
			}

			data = dec
		}

		env = models.LoadEnvironment(data)
	}

	r.ContainerDefinitions = make([]*ecs.ContainerDefinition, len(tasks))

	for i, itask := range tasks {
		task := itask.(map[string]interface{})

		cpu := 0
		var err error

		if c, ok := task["Cpu"].(string); ok && c != "" {
			cpu, err = strconv.Atoi(c)
			if err != nil {
				return "invalid", nil, err
			}
		}

		memory, err := strconv.Atoi(task["Memory"].(string))
		if err != nil {
			return "invalid", nil, err
		}

		privileged := false

		if p, ok := task["Privileged"].(string); ok && p != "" {
			privileged, err = strconv.ParseBool(p)
			if err != nil {
				return "invalid", nil, err
			}
		}

		r.ContainerDefinitions[i] = &ecs.ContainerDefinition{
			Name:       aws.String(task["Name"].(string)),
			Essential:  aws.Bool(true),
			Image:      aws.String(task["Image"].(string)),
			Cpu:        aws.Int64(int64(cpu)),
			Memory:     aws.Int64(int64(memory)),
			Privileged: aws.Bool(privileged),
		}

		// set Command from either -
		// a single string (shell form) - ["sh", "-c", command]
		// an array of strings (exec form) - ["cmd1", "cmd2"]
		switch commands := task["Command"].(type) {
		case string:
			if commands != "" {
				r.ContainerDefinitions[i].Command = []*string{aws.String("sh"), aws.String("-c"), aws.String(commands)}
			}
		case []interface{}:
			r.ContainerDefinitions[i].Command = make([]*string, len(commands))
			for j, command := range commands {
				r.ContainerDefinitions[i].Command[j] = aws.String(command.(string))
			}
		}

		// set Task environment from CF Tasks[].Environment key/values
		// These key/values are read from the app manifest environment hash
		if oenv, ok := task["Environment"].(map[string]interface{}); ok {
			for key, val := range oenv {
				r.ContainerDefinitions[i].Environment = append(r.ContainerDefinitions[i].Environment, &ecs.KeyValuePair{
					Name:  aws.String(key),
					Value: aws.String(val.(string)),
				})
			}
		}

		// set Task environment from decrypted S3 URL body of key/values
		// These key/values take precident over the above environment
		for key, val := range env {
			r.ContainerDefinitions[i].Environment = append(r.ContainerDefinitions[i].Environment, &ecs.KeyValuePair{
				Name:  aws.String(key),
				Value: aws.String(val),
			})
		}

		// set Release value in Task environment
		if release, ok := req.ResourceProperties["Release"].(string); ok {
			r.ContainerDefinitions[i].Environment = append(r.ContainerDefinitions[i].Environment, &ecs.KeyValuePair{
				Name:  aws.String("RELEASE"),
				Value: aws.String(release),
			})
		}

		// set links
		if links, ok := task["Links"].([]interface{}); ok {
			r.ContainerDefinitions[i].Links = make([]*string, len(links))

			for j, link := range links {
				r.ContainerDefinitions[i].Links[j] = aws.String(link.(string))
			}
		}

		// set portmappings
		if ports, ok := task["PortMappings"].([]interface{}); ok {

			r.ContainerDefinitions[i].PortMappings = make([]*ecs.PortMapping, len(ports))

			for j, port := range ports {
				parts := strings.Split(port.(string), ":")
				host, _ := strconv.Atoi(parts[0])
				container, _ := strconv.Atoi(parts[1])

				r.ContainerDefinitions[i].PortMappings[j] = &ecs.PortMapping{
					ContainerPort: aws.Int64(int64(container)),
					HostPort:      aws.Int64(int64(host)),
				}
			}
		}

		// set volumes
		if volumes, ok := task["Volumes"].([]interface{}); ok {
			for j, volume := range volumes {
				name := fmt.Sprintf("%s-%d-%d", task["Name"].(string), i, j)
				parts := strings.Split(volume.(string), ":")

				r.Volumes = append(r.Volumes, &ecs.Volume{
					Name: aws.String(name),
					Host: &ecs.HostVolumeProperties{
						SourcePath: aws.String(parts[0]),
					},
				})

				r.ContainerDefinitions[i].MountPoints = append(r.ContainerDefinitions[i].MountPoints, &ecs.MountPoint{
					SourceVolume:  aws.String(name),
					ContainerPath: aws.String(parts[1]),
					ReadOnly:      aws.Bool(false),
				})
			}
		}

		// set extra hosts
		if extraHosts, ok := task["ExtraHosts"].([]interface{}); ok {
			for _, host := range extraHosts {
				hostx, oky := host.(map[string]interface{})
				if oky {
					r.ContainerDefinitions[i].ExtraHosts = append(r.ContainerDefinitions[i].ExtraHosts, &ecs.HostEntry{
						Hostname:  aws.String(hostx["HostName"].(string)),
						IpAddress: aws.String(hostx["IpAddress"].(string)),
					})
				}
			}
		}
	}

	res, err := ECS(req).RegisterTaskDefinition(r)
	if err != nil {
		return "invalid", nil, err
	}

	return *res.TaskDefinition.TaskDefinitionArn, nil, nil
}

func ECSTaskDefinitionDelete(req Request) (string, map[string]string, error) {
	// We have observed a race condition quickly deregistering then re-registering
	// Task Definitions, where the Register fails. We work around this by not
	// deregistering any Task Definitions.
	// _, err := ECS(req).DeregisterTaskDefinition(&ecs.DeregisterTaskDefinitionInput{TaskDefinition: aws.String(req.PhysicalResourceId)})
	return req.PhysicalResourceId, nil, nil
}

var idAlphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func generateId(prefix string, size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = idAlphabet[rand.Intn(len(idAlphabet))]
	}
	return prefix + string(b)
}
