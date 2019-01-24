package k8s

import (
	"fmt"
	"io"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
	ac "k8s.io/api/core/v1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *Provider) InstanceKeyroll() error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) InstanceList() (structs.Instances, error) {
	ns, err := p.Cluster.CoreV1().Nodes().List(am.ListOptions{})
	if err != nil {
		return nil, err
	}

	// ms, err := p.Metrics.Metrics().NodeMetricses().List(am.ListOptions{})
	// if err != nil {
	//   return nil, err
	// }

	// fmt.Printf("ms = %+v\n", ms)

	is := structs.Instances{}

	for _, n := range ns.Items {
		pds, err := p.Cluster.CoreV1().Pods("").List(am.ListOptions{FieldSelector: fmt.Sprintf("spec.nodeName=%s", n.ObjectMeta.Name)})
		if err != nil {
			return nil, err
		}

		status := "pending"

		for _, c := range n.Status.Conditions {
			if c.Type == "Ready" && c.Status == "True" {
				status = "running"
			}
		}

		private := ""
		public := ""

		for _, na := range n.Status.Addresses {
			switch na.Type {
			case ac.NodeExternalIP:
				public = na.Address
			case ac.NodeInternalIP:
				private = na.Address
			}
		}

		is = append(is, structs.Instance{
			Id:        helpers.CoalesceString(n.Spec.ProviderID, n.ObjectMeta.Name),
			PrivateIp: private,
			Processes: len(pds.Items),
			PublicIp:  public,
			Started:   n.CreationTimestamp.Time,
			Status:    status,
		})
	}

	return is, nil
}

func (p *Provider) InstanceShell(id string, rw io.ReadWriter, opts structs.InstanceShellOptions) (int, error) {
	return 0, fmt.Errorf("unimplemented")
}

func (p *Provider) InstanceTerminate(id string) error {
	return fmt.Errorf("unimplemented")
}
