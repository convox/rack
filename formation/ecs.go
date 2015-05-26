package formation

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ecs"
)

func HandleECSService(req Request) (string, error) {
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

	return "", fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

func HandleECSTaskDefinition(req Request) (string, error) {
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

	return "", fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

func ECSServiceCreate(req Request) (string, error) {
	count, err := strconv.Atoi(req.ResourceProperties["DesiredCount"].(string))

	if err != nil {
		return "", err
	}

	r := &ecs.CreateServiceInput{
		Cluster:        aws.String(req.ResourceProperties["Cluster"].(string)),
		DesiredCount:   aws.Long(int64(count)),
		ServiceName:    aws.String(req.ResourceProperties["Name"].(string)),
		TaskDefinition: aws.String(req.ResourceProperties["TaskDefinition"].(string)),
	}

	balancers := req.ResourceProperties["LoadBalancers"].([]interface{})

	if len(balancers) > 0 {
		r.Role = aws.String(req.ResourceProperties["Role"].(string))
	}

	r.LoadBalancers = make([]*ecs.LoadBalancer, len(balancers))

	for i, balancer := range balancers {
		parts := strings.Split(balancer.(string), ":")
		name := parts[0]
		port, _ := strconv.Atoi(parts[1])
		r.LoadBalancers[i] = &ecs.LoadBalancer{
			ContainerName:    aws.String("main"),
			LoadBalancerName: aws.String(name),
			ContainerPort:    aws.Long(int64(port)),
		}
	}

	res, err := ECS().CreateService(r)

	if err != nil {
		return "", err
	}

	return *res.Service.ServiceARN, nil
}

func ECSServiceUpdate(req Request) (string, error) {
	count, _ := strconv.Atoi(req.ResourceProperties["DesiredCount"].(string))

	res, err := ECS().UpdateService(&ecs.UpdateServiceInput{
		Cluster:        aws.String(req.ResourceProperties["Cluster"].(string)),
		Service:        aws.String(req.ResourceProperties["Name"].(string)),
		DesiredCount:   aws.Long(int64(count)),
		TaskDefinition: aws.String(req.ResourceProperties["TaskDefinition"].(string)),
	})

	if err != nil {
		return "", err
	}

	return *res.Service.ServiceARN, nil
}

func ECSServiceDelete(req Request) (string, error) {
	cluster := req.ResourceProperties["Cluster"].(string)
	name := req.ResourceProperties["Name"].(string)

	_, err := ECS().UpdateService(&ecs.UpdateServiceInput{
		Cluster:      aws.String(cluster),
		Service:      aws.String(name),
		DesiredCount: aws.Long(0),
	})

	// go ahead and mark the delete good if the service is not found
	if ae, ok := err.(aws.APIError); ok {
		if ae.Code == "ServiceNotFoundException" {
			return "", nil
		}
	}

	if err != nil {
		return "", err
	}

	_, err = ECS().DeleteService(&ecs.DeleteServiceInput{
		Cluster: aws.String(cluster),
		Service: aws.String(name),
	})

	// TODO let the cloudformation finish thinking this deleted
	// but take note so we can figure out why
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return "", nil
	}

	return "", nil
}

func ECSTaskDefinitionCreate(req Request) (string, error) {
	cpu, _ := strconv.Atoi(req.ResourceProperties["CPU"].(string))
	memory, _ := strconv.Atoi(req.ResourceProperties["Memory"].(string))

	r := &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{
			{
				CPU:       aws.Long(int64(cpu)),
				Essential: aws.Boolean(true),
				Image:     aws.String(req.ResourceProperties["Image"].(string)),
				Memory:    aws.Long(int64(memory)),
				Name:      aws.String("main"),
			},
		},
		Family: aws.String(req.ResourceProperties["Name"].(string)),
	}

	if command := req.ResourceProperties["Command"].(string); command != "" {
		r.ContainerDefinitions[0].Command = []*string{aws.String("sh"), aws.String("-c"), aws.String(command)}
	}

	// set environment
	// env := LoadEnvironment([]byte(r.Env))

	// for key, val := range env {
	//   req.ContainerDefinitions[0].Environment = append(req.ContainerDefinitions[0].Environment, &ecs.KeyValuePair{
	//     Name:  aws.String(key),
	//     Value: aws.String(val),
	//   })
	// }

	// set portmappings
	ports := req.ResourceProperties["PortMappings"].([]interface{})

	r.ContainerDefinitions[0].PortMappings = make([]*ecs.PortMapping, len(ports))

	for i, port := range ports {
		parts := strings.Split(port.(string), ":")
		host, _ := strconv.Atoi(parts[0])
		container, _ := strconv.Atoi(parts[1])

		r.ContainerDefinitions[0].PortMappings[i] = &ecs.PortMapping{
			ContainerPort: aws.Long(int64(container)),
			HostPort:      aws.Long(int64(host)),
		}
	}

	res, err := ECS().RegisterTaskDefinition(r)

	if err != nil {
		return "", err
	}

	return *res.TaskDefinition.TaskDefinitionARN, nil
}

func ECSTaskDefinitionDelete(req Request) (string, error) {
	// TODO: currently unsupported by ECS
	// res, err := ECS().DeregisterTaskDefinition(&ecs.DeregisterTaskDefinitionInput{TaskDefinition: aws.String(req.PhysicalResourceId)})
	return "", nil
}
