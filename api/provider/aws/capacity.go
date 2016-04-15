package aws

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/api/structs"
)

// returns individual server memory, total rack memory
func (p *AWSProvider) CapacityGet() (*structs.Capacity, error) {
	capacity := &structs.Capacity{}

	lres, err := p.ecs().ListContainerInstances(&ecs.ListContainerInstancesInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
	})

	if err != nil {
		return nil, err
	}

	ires, err := p.ecs().DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(os.Getenv("CLUSTER")),
		ContainerInstances: lres.ContainerInstanceArns,
	})

	if err != nil {
		return nil, err
	}

	for _, instance := range ires.ContainerInstances {
		for _, resource := range instance.RegisteredResources {
			if *resource.Name == "MEMORY" {
				capacity.InstanceMemory = *resource.IntegerValue
				capacity.ClusterMemory += *resource.IntegerValue
				break
			}
		}
	}

	services, err := p.clusterServices()

	if err != nil {
		return nil, err
	}

	for _, service := range services {
		if len(service.LoadBalancers) > 0 && *service.DesiredCount > capacity.ProcessWidth {
			capacity.ProcessWidth = *service.DesiredCount
		}

		res, err := p.ecs().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: service.TaskDefinition,
		})

		if err != nil {
			return nil, err
		}

		for _, cd := range res.TaskDefinition.ContainerDefinitions {
			capacity.ProcessCount += *service.DesiredCount
			capacity.ProcessMemory += (*service.DesiredCount * *cd.Memory)
		}
	}

	// return capacity, concurrency, nil
	return capacity, nil
}

type ECSServices []*ecs.Service

func (p *AWSProvider) clusterServices() (ECSServices, error) {
	services := ECSServices{}

	lsres, err := p.ecs().ListServices(&ecs.ListServicesInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
	})

	if err != nil {
		return services, err
	}

	dsres, err := p.ecs().DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(os.Getenv("CLUSTER")),
		Services: lsres.ServiceArns,
	})

	if err != nil {
		return services, err
	}

	for i := 0; i < len(dsres.Services); i++ {
		services = append(services, dsres.Services[i])
	}

	return services, nil
}
