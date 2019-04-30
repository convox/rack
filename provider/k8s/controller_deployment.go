package k8s

import (
	"fmt"
	"time"

	"github.com/convox/rack/pkg/kctl"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	ac "k8s.io/api/core/v1"
	ae "k8s.io/api/extensions/v1beta1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	ic "k8s.io/client-go/informers/extensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type DeploymentController struct {
	Controller *kctl.Controller
	Provider   *Provider
}

func NewDeploymentController(p *Provider) (*DeploymentController, error) {
	pc := &DeploymentController{
		Provider: p,
	}

	c, err := kctl.NewController(p.Rack, "convox-k8s-deployment", pc)
	if err != nil {
		return nil, err
	}

	pc.Controller = c

	return pc, nil
}

func (c *DeploymentController) Client() kubernetes.Interface {
	return c.Provider.Cluster
}

func (c *DeploymentController) ListOptions(opts *am.ListOptions) {
	// opts.LabelSelector = fmt.Sprintf("system=convox,rack=%s", c.Provider.Rack)
}

func (c *DeploymentController) Run() {
	i := ic.NewFilteredDeploymentInformer(c.Provider.Cluster, ac.NamespaceAll, 0, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, c.ListOptions)

	ch := make(chan error)

	go c.Controller.Run(i, ch)

	for err := range ch {
		fmt.Printf("err = %+v\n", err)
	}
}

func (c *DeploymentController) Start() error {
	return nil
}

func (c *DeploymentController) Stop() error {
	return nil
}

func (c *DeploymentController) Add(obj interface{}) error {
	// d, err := assertDeployment(obj)
	// if err != nil {
	//   return err
	// }

	// fmt.Printf("deployment add: %s/%s\n", d.ObjectMeta.Namespace, d.ObjectMeta.Name)

	return nil
}

func (c *DeploymentController) Delete(obj interface{}) error {
	// d, err := assertDeployment(obj)
	// if err != nil {
	//   return err
	// }

	// fmt.Printf("deployment delete: %s/%s\n", d.ObjectMeta.Namespace, d.ObjectMeta.Name)

	return nil
}

func (c *DeploymentController) Update(prev, cur interface{}) error {
	pd, err := assertDeployment(prev)
	if err != nil {
		return err
	}

	cd, err := assertDeployment(cur)
	if err != nil {
		return err
	}

	if pd.ResourceVersion == cd.ResourceVersion {
		return nil
	}

	fmt.Printf("deployment update: %s/%s\n", cd.ObjectMeta.Namespace, cd.ObjectMeta.Name)

	if deploymentConditionReason(pd, "Progressing") != "NewReplicaSetAvailable" && deploymentConditionReason(cd, "Progressing") == "NewReplicaSetAvailable" {
		if cd.ObjectMeta.Labels["app"] != "" && cd.ObjectMeta.Labels["release"] != "" {
			if err := c.Provider.serviceInstall(cd.ObjectMeta.Labels["app"], cd.ObjectMeta.Labels["release"], cd.Name); err != nil {
				return err
			}
		}
	}

	// if deploymentCondition(cd, "Progressing") == "False" && deploymentCondition(pd, "Progressing") == "True" && deploymentConditionReason(cd, "Progressing") == "ProgressDeadlineExceeded" {
	//   if err := c.deploymentRollback(cd); err != nil {
	//     return err
	//   }
	// }

	// if deploymentCondition(cd, "Rollback") == "True" && cd.Status.UnavailableReplicas == 0 {
	//   go func() {
	//     time.Sleep(10 * time.Second)
	//     c.setDeploymentCondition(cd, "Rollback", "False", "")
	//   }()
	// }

	return nil
}

func (c *DeploymentController) deploymentLog(d *ae.Deployment, message string) error {
	return c.Provider.systemLog(d.ObjectMeta.Labels["app"], d.Name, time.Now().UTC(), message)
}

func (c *DeploymentController) deploymentRollback(d *ae.Deployment) error {
	app := d.ObjectMeta.Labels["app"]
	if app == "" {
		return nil
	}

	c.deploymentLog(d, "Promotion timeout")

	if rollback := d.ObjectMeta.Annotations["convox.rollback"]; rollback != "" {
		c.deploymentLog(d, fmt.Sprintf("Rolling back to %s", rollback))

		if err := c.Provider.Engine.ReleasePromote(app, rollback, structs.ReleasePromoteOptions{}); err != nil {
			fmt.Printf("err = %+v\n", err)
			return err
		}
	} else {
		c.deploymentLog(d, "Rolling back")

		if err := c.deploymentScale(d, 0); err != nil {
			fmt.Printf("err = %+v\n", err)
			return err
		}
	}

	if err := c.setDeploymentCondition(d, "Rollback", "True", "ProgressDeadlineExceeded"); err != nil {
		return err
	}

	return nil
}

func (c *DeploymentController) deploymentScale(d *ae.Deployment, count int32) error {
	ud, err := c.Provider.Cluster.ExtensionsV1beta1().Deployments(d.ObjectMeta.Namespace).Get(d.ObjectMeta.Name, am.GetOptions{})
	if err != nil {
		return err
	}

	ud.Spec.Replicas = options.Int32(count)

	if _, err := c.Client().ExtensionsV1beta1().Deployments(d.ObjectMeta.Namespace).Update(ud); err != nil {
		return err
	}

	return nil
}

func (c *DeploymentController) setDeploymentCondition(d *ae.Deployment, name, status, reason string) error {
	ud, err := c.Provider.Cluster.ExtensionsV1beta1().Deployments(d.ObjectMeta.Namespace).Get(d.ObjectMeta.Name, am.GetOptions{ResourceVersion: d.ResourceVersion})
	if err != nil {
		return err
	}

	found := false

	for i, sc := range ud.Status.Conditions {
		if sc.Type == ae.DeploymentConditionType(name) {
			ud.Status.Conditions[i].Reason = reason
			ud.Status.Conditions[i].Status = ac.ConditionStatus(status)
			found = true
			break
		}
	}

	if !found {
		ud.Status.Conditions = append(ud.Status.Conditions, ae.DeploymentCondition{
			Reason: reason,
			Status: ac.ConditionStatus(status),
			Type:   ae.DeploymentConditionType(name),
		})
	}

	if _, err := c.Provider.Cluster.ExtensionsV1beta1().Deployments(ud.ObjectMeta.Namespace).UpdateStatus(ud); err != nil {
		return err
	}

	return nil
}

func assertDeployment(v interface{}) (*ae.Deployment, error) {
	d, ok := v.(*ae.Deployment)
	if !ok {
		return nil, fmt.Errorf("could not assert deployment for type: %T", v)
	}

	return d, nil
}

func deploymentCondition(d *ae.Deployment, name string) string {
	for _, c := range d.Status.Conditions {
		if string(c.Type) == name {
			return string(c.Status)
		}
	}

	return ""
}

func deploymentConditionReason(d *ae.Deployment, name string) string {
	for _, c := range d.Status.Conditions {
		if string(c.Type) == name {
			return c.Reason
		}
	}

	return ""
}
