package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ecs"
)

type clearMonitoringMetric struct {
	_                 struct{}  `type:"structure"`
	MetricNames       []*string `locationName:"metricNames" type:"list"`
	ResolutionSeconds *int64    `locationName:"resolutionSeconds" type:"integer"`
}

type clearMonitoringConfig struct {
	_                    struct{}                 `type:"structure"`
	MetricConfigurations []*clearMonitoringMetric `locationName:"metricConfigurations" type:"list"`
}

type clearMonitoringInput struct {
	_          struct{}               `type:"structure"`
	Cluster    *string                `locationName:"cluster" type:"string"`
	Service    *string                `locationName:"service" type:"string"`
	Monitoring *clearMonitoringConfig `locationName:"monitoring" type:"structure"`
}

// clearGen1ServiceMonitoring clears the ECS monitoring config on a gen1 app's Classic-LB services
// only (target-group services are left untouched). AWS rejects UpdateService for a Classic-LB service
// that carries a monitoring config; an empty metricConfigurations removes it. Best-effort.
func (p *Provider) clearGen1ServiceMonitoring(app string) {
	srs, err := p.listStackResources(p.rackStack(app))
	if err != nil {
		fmt.Printf("ns=kernel at=release.promote at=clearServiceMonitoring app=%q err=%q\n", app, err.Error())
		return
	}

	for _, batch := range chunkStrings(ecsServiceNamesInStack(srs), 10) {
		out, err := p.ecs().DescribeServices(&ecs.DescribeServicesInput{
			Cluster:  aws.String(p.Cluster),
			Services: aws.StringSlice(batch),
		})
		if err != nil {
			fmt.Printf("ns=kernel at=release.promote at=clearServiceMonitoring app=%q err=%q\n", app, err.Error())
			continue
		}

		for _, svc := range out.Services {
			if !serviceHasClassicLB(svc) {
				continue
			}

			name := aws.StringValue(svc.ServiceName)

			op := &request.Operation{Name: "UpdateService", HTTPMethod: "POST", HTTPPath: "/"}

			in := &clearMonitoringInput{
				Cluster:    aws.String(p.Cluster),
				Service:    aws.String(name),
				Monitoring: &clearMonitoringConfig{MetricConfigurations: []*clearMonitoringMetric{}},
			}

			errStr := ""
			if err := p.ecs().NewRequest(op, in, &ecs.UpdateServiceOutput{}).Send(); err != nil {
				errStr = err.Error()
			}

			fmt.Printf("ns=kernel at=release.promote at=clearServiceMonitoring service=%q err=%q\n", name, errStr)
		}
	}
}

func serviceHasClassicLB(svc *ecs.Service) bool {
	if svc == nil {
		return false
	}
	for _, lb := range svc.LoadBalancers {
		if lb != nil && lb.LoadBalancerName != nil && *lb.LoadBalancerName != "" {
			return true
		}
	}
	return false
}

func ecsServiceNamesInStack(srs []*cloudformation.StackResourceSummary) []string {
	names := []string{}

	for _, sr := range srs {
		if sr.ResourceType == nil || sr.PhysicalResourceId == nil {
			continue
		}

		switch *sr.ResourceType {
		case "AWS::ECS::Service", "Custom::ECSService":
		default:
			continue
		}

		names = append(names, ecsServiceName(*sr.PhysicalResourceId))
	}

	return names
}

func ecsServiceName(physicalID string) string {
	if i := strings.LastIndex(physicalID, "/"); i >= 0 {
		return physicalID[i+1:]
	}
	return physicalID
}

func chunkStrings(s []string, n int) [][]string {
	chunks := [][]string{}

	for i := 0; i < len(s); i += n {
		end := i + n
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[i:end])
	}

	return chunks
}
