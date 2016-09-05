package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/api/structs"
)

// returns individual server memory, total rack memory
func (p *AWSProvider) CapacityGet() (*structs.Capacity, error) {
	log := Logger.At("CapacityGet").Start()

	capacity := &structs.Capacity{}

	ires, err := p.describeContainerInstances()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	for _, instance := range ires.ContainerInstances {
		for _, resource := range instance.RegisteredResources {
			if *resource.Name == "MEMORY" {
				capacity.InstanceMemory = *resource.IntegerValue
				capacity.ClusterMemory += *resource.IntegerValue
			}
			if *resource.Name == "CPU" {
				capacity.InstanceCPU = *resource.IntegerValue
				capacity.ClusterCPU += *resource.IntegerValue
			}
		}
	}

	services, err := p.clusterServices()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	portWidth := map[int64]int64{}

	for _, service := range services {
		if len(service.LoadBalancers) > 0 {
			for _, deployment := range service.Deployments {
				td, err := p.describeTaskDefinition(*deployment.TaskDefinition)
				if err != nil {
					log.Error(err)
					return nil, err
				}

				tdPorts := map[string]int64{}

				for _, cd := range td.ContainerDefinitions {
					for _, pm := range cd.PortMappings {
						tdPorts[fmt.Sprintf("%s.%d", *cd.Name, *pm.ContainerPort)] = *pm.HostPort
					}
				}

				for _, lb := range service.LoadBalancers {
					if port, ok := tdPorts[fmt.Sprintf("%s.%d", *lb.ContainerName, *lb.ContainerPort)]; ok {
						portWidth[port] += *deployment.DesiredCount
					}
				}
			}
		}

		res, err := p.ecs().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: service.TaskDefinition,
		})
		if err != nil {
			log.Error(err)
			return nil, err
		}

		for _, cd := range res.TaskDefinition.ContainerDefinitions {
			capacity.ProcessCount += *service.DesiredCount
			capacity.ProcessMemory += (*service.DesiredCount * *cd.Memory)
			capacity.ProcessCPU += (*service.DesiredCount * *cd.Cpu)
		}
	}

	max := int64(0)

	for _, n := range portWidth {
		if n > max {
			max = n
		}
	}

	capacity.ProcessWidth = max

	log.Success()
	// "cluster.cpu=%d cluster.memory=%d instance.cpu=%d instance.memory=%d process.count=%d process.cpu=%d process.memory=%d process.width=%d", capacity.ClusterCPU, capacity.ClusterMemory, capacity.InstanceCPU, capacity.InstanceMemory, capacity.ProcessCount, capacity.ProcessCPU, capacity.ProcessMemory, capacity.ProcessWidth)
	return capacity, nil
}

type ECSServices []*ecs.Service

func (p *AWSProvider) clusterServices() (ECSServices, error) {
	services := ECSServices{}

	lsres, err := p.ecs().ListServices(&ecs.ListServicesInput{
		Cluster: aws.String(p.Cluster),
	})

	if err != nil {
		return services, err
	}

	dsres, err := p.ecs().DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(p.Cluster),
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
