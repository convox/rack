package handler

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/api/crypt"
	"github.com/convox/rack/api/models"
)

func HandleECSCluster(req Request) (string, map[string]string, error) {
	defer recoverFailure(req)

	switch req.RequestType {
	case "Create":
		fmt.Println("CREATING CLUSTER")
		fmt.Printf("req %+v\n", req)
		return ECSClusterCreate(req)
	case "Update":
		fmt.Println("UPDATING CLUSTER")
		fmt.Printf("req %+v\n", req)
		return ECSClusterUpdate(req)
	case "Delete":
		fmt.Println("DELETING CLUSTER")
		fmt.Printf("req %+v\n", req)
		return ECSClusterDelete(req)
	}

	return "", nil, fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

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

	return "", nil, fmt.Errorf("unknown RequestType: %s", req.RequestType)
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

	return "", nil, fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

func ECSClusterCreate(req Request) (string, map[string]string, error) {
	res, err := ECS(req).CreateCluster(&ecs.CreateClusterInput{
		ClusterName: aws.String(req.ResourceProperties["Name"].(string)),
	})

	if err != nil {
		return "", nil, err
	}

	return *res.Cluster.ClusterARN, nil, nil
}

func ECSClusterUpdate(req Request) (string, map[string]string, error) {
	return req.PhysicalResourceId, nil, fmt.Errorf("could not update")
}

func ECSClusterDelete(req Request) (string, map[string]string, error) {
	_, err := ECS(req).DeleteCluster(&ecs.DeleteClusterInput{
		Cluster: aws.String(req.PhysicalResourceId),
	})

	// TODO let the cloudformation finish thinking this deleted
	// but take note so we can figure out why
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return req.PhysicalResourceId, nil, nil
	}

	return req.PhysicalResourceId, nil, nil
}

func ECSServiceCreate(req Request) (string, map[string]string, error) {
	count, err := strconv.Atoi(req.ResourceProperties["DesiredCount"].(string))

	if err != nil {
		return "", nil, err
	}

	r := &ecs.CreateServiceInput{
		Cluster:        aws.String(req.ResourceProperties["Cluster"].(string)),
		DesiredCount:   aws.Long(int64(count)),
		ServiceName:    aws.String(req.ResourceProperties["Name"].(string) + "-" + generateId("S", 10)),
		TaskDefinition: aws.String(req.ResourceProperties["TaskDefinition"].(string)),
	}

	balancers := req.ResourceProperties["LoadBalancers"].([]interface{})

	if len(balancers) > 0 {
		r.Role = aws.String(req.ResourceProperties["Role"].(string))
	}

	for _, balancer := range balancers {
		parts := strings.SplitN(balancer.(string), ":", 3)

		if len(parts) != 3 {
			return "", nil, fmt.Errorf("invalid load balancer specification: %s", balancer.(string))
		}

		name := parts[0]
		ps := parts[1]
		port, _ := strconv.Atoi(parts[2])

		r.LoadBalancers = append(r.LoadBalancers, &ecs.LoadBalancer{
			LoadBalancerName: aws.String(name),
			ContainerName:    aws.String(ps),
			ContainerPort:    aws.Long(int64(port)),
		})

		break
	}

	res, err := ECS(req).CreateService(r)

	if err != nil {
		return "", nil, err
	}

	return *res.Service.ServiceARN, nil, nil
}

func ECSServiceUpdate(req Request) (string, map[string]string, error) {
	count, _ := strconv.Atoi(req.ResourceProperties["DesiredCount"].(string))

	// arn:aws:ecs:us-east-1:922560784203:service/sinatra-SZXTRXEMYEY
	parts := strings.Split(req.PhysicalResourceId, "/")
	name := parts[1]

	res, err := ECS(req).UpdateService(&ecs.UpdateServiceInput{
		Cluster:        aws.String(req.ResourceProperties["Cluster"].(string)),
		Service:        aws.String(name),
		DesiredCount:   aws.Long(int64(count)),
		TaskDefinition: aws.String(req.ResourceProperties["TaskDefinition"].(string)),
	})

	if err != nil {
		return "", nil, err
	}

	return *res.Service.ServiceARN, nil, nil
}

func ECSServiceDelete(req Request) (string, map[string]string, error) {
	cluster := req.ResourceProperties["Cluster"].(string)

	// arn:aws:ecs:us-east-1:922560784203:service/sinatra-SZXTRXEMYEY
	parts := strings.Split(req.PhysicalResourceId, "/")
	name := parts[1]

	_, err := ECS(req).UpdateService(&ecs.UpdateServiceInput{
		Cluster:      aws.String(cluster),
		Service:      aws.String(name),
		DesiredCount: aws.Long(0),
	})

	// go ahead and mark the delete good if the service is not found
	if ae, ok := err.(awserr.Error); ok {
		if ae.Code() == "ServiceNotFoundException" {
			return req.PhysicalResourceId, nil, nil
		}
	}

	// TODO let the cloudformation finish thinking this deleted
	// but take note so we can figure out why
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return req.PhysicalResourceId, nil, nil
	}

	_, err = ECS(req).DeleteService(&ecs.DeleteServiceInput{
		Cluster: aws.String(cluster),
		Service: aws.String(name),
	})

	// TODO let the cloudformation finish thinking this deleted
	// but take note so we can figure out why
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return req.PhysicalResourceId, nil, nil
	}

	return req.PhysicalResourceId, nil, nil
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
			return "", nil, err
		}

		defer res.Body.Close()

		data, err := ioutil.ReadAll(res.Body)

		if key, ok := req.ResourceProperties["Key"].(string); ok && key != "" {
			cr := crypt.New(Region(&req), os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"))
			cr.AwsToken = os.Getenv("AWS_SESSION_TOKEN")

			dec, err := cr.Decrypt(key, data)

			if err != nil {
				return "", nil, err
			}

			data = dec
		}

		env = models.LoadEnvironment(data)
	}

	r.ContainerDefinitions = make([]*ecs.ContainerDefinition, len(tasks))

	for i, itask := range tasks {
		task := itask.(map[string]interface{})

		cpu, _ := strconv.Atoi(task["CPU"].(string))
		memory, _ := strconv.Atoi(task["Memory"].(string))

		r.ContainerDefinitions[i] = &ecs.ContainerDefinition{
			Name:      aws.String(task["Name"].(string)),
			Essential: aws.Boolean(true),
			Image:     aws.String(task["Image"].(string)),
			CPU:       aws.Long(int64(cpu)),
			Memory:    aws.Long(int64(memory)),
		}

		if command, ok := task["Command"].(string); ok && command != "" {
			r.ContainerDefinitions[i].Command = []*string{aws.String("sh"), aws.String("-c"), aws.String(command)}
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
					ContainerPort: aws.Long(int64(container)),
					HostPort:      aws.Long(int64(host)),
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
					ReadOnly:      aws.Boolean(false),
				})
			}
		}
	}

	res, err := ECS(req).RegisterTaskDefinition(r)

	if err != nil {
		return "", nil, err
	}

	return *res.TaskDefinition.TaskDefinitionARN, nil, nil
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
