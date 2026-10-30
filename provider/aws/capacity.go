package aws

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/pkg/cache"
	"github.com/convox/rack/pkg/structs"
)

// CapacityGet returns individual server and total rack resources
func (p *Provider) CapacityGet() (*structs.Capacity, error) {
	log := Logger.At("CapacityGet").Start()

	capacity := &structs.Capacity{}

	ires, err := p.listAndDescribeContainerInstances()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	for _, instance := range ires.ContainerInstances {
		if *instance.Status == "DRAINING" {
			continue
		}

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
		servicePortWidth := map[int64]int64{}

		if service.LaunchType != nil && *service.LaunchType != "EC2" {
			continue
		}

		if len(service.LoadBalancers) > 0 {
			for _, deployment := range service.Deployments {
				res, err := p.describeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
					TaskDefinition: deployment.TaskDefinition,
				})
				if err != nil {
					log.Error(err)
					return nil, err
				}

				tdPorts := map[string]int64{}

				for _, cd := range res.TaskDefinition.ContainerDefinitions {
					if g := cd.DockerLabels["convox.generation"]; g == nil || *g != "2" {
						for _, pm := range cd.PortMappings {
							tdPorts[fmt.Sprintf("%s.%d", *cd.Name, *pm.ContainerPort)] = *pm.HostPort
						}
					}
				}

				for _, lb := range service.LoadBalancers {
					if port, ok := tdPorts[fmt.Sprintf("%s.%d", *lb.ContainerName, *lb.ContainerPort)]; ok {
						servicePortWidth[port] += *deployment.DesiredCount
					}
				}
			}
		}

		// take deploymentconfiguration into account during deploys
		if len(service.Deployments) > 1 {
			if dc := service.DeploymentConfiguration; dc != nil {
				if mp := dc.MinimumHealthyPercent; mp != nil {
					mult := float64(*mp) / float64(100)

					for port, width := range servicePortWidth {
						servicePortWidth[port] = int64(math.Ceil(float64(width) * mult))
					}
				}
			}
		}

		for port, width := range servicePortWidth {
			portWidth[port] += width
		}

		res, err := p.describeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: service.TaskDefinition,
		})
		if err != nil {
			log.Error(err)
			return nil, err
		}

		if service.DesiredCount == nil {
			return nil, fmt.Errorf("invalid DesiredCount")
		}

		minCount := *service.DesiredCount

		for _, cd := range res.TaskDefinition.ContainerDefinitions {
			capacity.ProcessCount += minCount

			if cd.Memory != nil {
				capacity.ProcessMemory += (minCount * *cd.Memory)
			}

			if cd.Cpu != nil {
				capacity.ProcessCPU += (minCount * *cd.Cpu)
			}
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

func (p *Provider) clusterServices() (ECSServices, error) {
	v, err := cache.Fetch("clusterServices", p.Cluster, 5*time.Second, func() (interface{}, bool, error) {
		arns := []string{}

		err := p.ecs().ListServicesPages(&ecs.ListServicesInput{
			Cluster:    aws.String(p.Cluster),
			MaxResults: aws.Int64(100),
		}, func(page *ecs.ListServicesOutput, lastPage bool) bool {
			arns = append(arns, aws.StringValueSlice(page.ServiceArns)...)
			return true
		})
		if err != nil {
			return nil, false, err
		}

		services, err := p.describeServicesByID(arns)

		return services, !p.SkipCache, err
	})
	if err != nil {
		return nil, err
	}

	services, _ := v.(ECSServices)

	return services, nil
}

const describeServicesPageSize = 10

func (p *Provider) describeServicesByID(ids []string) (ECSServices, error) {
	ids = slices.Clone(ids)
	sort.Strings(ids)

	services := ECSServices{}

	for i := 0; i < len(ids); i += describeServicesPageSize {
		res, err := p.describeServices(&ecs.DescribeServicesInput{
			Cluster:  aws.String(p.Cluster),
			Services: aws.StringSlice(ids[i:min(i+describeServicesPageSize, len(ids))]),
		})
		if err != nil {
			return nil, err
		}

		services = append(services, res.Services...)
	}

	return services, nil
}
