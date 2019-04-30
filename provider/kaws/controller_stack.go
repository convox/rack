package kaws

import (
	"fmt"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/kctl"
	ct "github.com/convox/rack/provider/kaws/pkg/apis/convox/v1"
	ic "github.com/convox/rack/provider/kaws/pkg/client/informers/externalversions/convox/v1"
	ac "k8s.io/api/core/v1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type StackController struct {
	Controller *kctl.Controller
	Provider   *Provider
}

func NewStackController(p *Provider) (*StackController, error) {
	pc := &StackController{
		Provider: p,
	}

	c, err := kctl.NewController(p.Rack, "convox-kaws-stack", pc)
	if err != nil {
		return nil, err
	}

	pc.Controller = c

	return pc, nil
}

func (c *StackController) Client() kubernetes.Interface {
	return c.Provider.Provider.Cluster
}

func (c *StackController) ListOptions(opts *am.ListOptions) {
	opts.LabelSelector = fmt.Sprintf("system=convox,rack=%s", c.Provider.Rack)
}

func (c *StackController) Run() {
	cc, err := c.Provider.convoxClient()
	if err != nil {
		fmt.Printf("err: %+v\n", err)
		return
	}

	i := ic.NewFilteredStackInformer(cc, ac.NamespaceAll, 1*time.Minute, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, c.ListOptions)

	ch := make(chan error)

	go c.Controller.Run(i, ch)

	for err := range ch {
		fmt.Printf("err = %+v\n", err)
	}
}

func (c *StackController) Start() error {
	return nil
}

func (c *StackController) Stop() error {
	return nil
}

func (c *StackController) Add(obj interface{}) error {
	s, err := assertStack(obj)
	if err != nil {
		return err
	}

	fmt.Printf("stack add: %s/%s\n", s.ObjectMeta.Namespace, s.ObjectMeta.Name)

	if err := c.stackSync(s); err != nil {
		return err
	}

	return nil
}

func (c *StackController) Delete(obj interface{}) error {
	return nil
}

func (c *StackController) Update(prev, cur interface{}) error {
	ps, err := assertStack(prev)
	if err != nil {
		return err
	}

	cs, err := assertStack(cur)
	if err != nil {
		return err
	}

	if cs.DeletionTimestamp != nil /*&& cs.DeletionTimestamp.Time.Before(time.Now().UTC().Add(-1*time.Minute))*/ && cs.Status != "Deleting" {
		// as, err := c.Provider.Provider.AppStatus(cs.ObjectMeta.Labels["app"])
		// if err != nil {
		//   return err
		// }

		// if as == "running" {
		return c.stackDelete(cs)
		// }
	}

	if reflect.DeepEqual(ps.ObjectMeta.Labels, cs.ObjectMeta.Labels) && reflect.DeepEqual(ps.Spec, cs.Spec) {
		return nil
	}

	fmt.Printf("stack update: %s/%s\n", cs.ObjectMeta.Name, cs.Status)

	if err := c.stackSync(cs); err != nil {
		return err
	}

	return nil
}

func (c *StackController) stackName(s *ct.Stack) string {
	return fmt.Sprintf("%s-%s", s.Namespace, s.Name)
}

func (c *StackController) stackSync(s *ct.Stack) error {
	if !s.DeletionTimestamp.IsZero() {
		return nil
	}

	_, err := c.Provider.CloudFormation.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(c.stackName(s)),
	})
	if helpers.AwsErrorCode(err) == "ValidationError" {
		return c.stackCreate(s)
	}

	return c.stackUpdate(s)
}

func (c *StackController) stackCreate(s *ct.Stack) error {
	req := &cloudformation.CreateStackInput{
		StackName:        aws.String(c.stackName(s)),
		TemplateBody:     aws.String(s.Spec.Template),
		NotificationARNs: []*string{aws.String(c.Provider.EventTopic)},
		Parameters:       []*cloudformation.Parameter{},
		Tags:             []*cloudformation.Tag{{Key: aws.String("stack"), Value: aws.String(s.ObjectMeta.Name)}},
	}

	for k, v := range s.Spec.Parameters {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(k),
			ParameterValue: aws.String(v),
		})
	}

	for k, v := range s.ObjectMeta.Labels {
		req.Tags = append(req.Tags, &cloudformation.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	if _, err := c.Provider.CloudFormation.CreateStack(req); err != nil {
		fmt.Printf("err: %+v\n", err)
		fmt.Printf("helpers.AwsErrorCode(err): %+v\n", helpers.AwsErrorCode(err))
		return err
	}

	return nil
}

func (c *StackController) stackDelete(s *ct.Stack) error {
	_, err := c.Provider.CloudFormation.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: aws.String(c.stackName(s)),
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *StackController) stackUpdate(s *ct.Stack) error {
	req := &cloudformation.UpdateStackInput{
		StackName:    aws.String(c.stackName(s)),
		TemplateBody: aws.String(s.Spec.Template),
		Parameters:   []*cloudformation.Parameter{},
		Tags:         []*cloudformation.Tag{{Key: aws.String("stack"), Value: aws.String(s.ObjectMeta.Name)}},
	}

	for k, v := range s.Spec.Parameters {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(k),
			ParameterValue: aws.String(v),
		})
	}

	for k, v := range s.ObjectMeta.Labels {
		req.Tags = append(req.Tags, &cloudformation.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	if _, err := c.Provider.CloudFormation.UpdateStack(req); err != nil {
		if !cloudformationErrorNoUpdates(err) {
			return err
		}

		if err := c.statusUpdate(s, "Running"); err != nil {
			return err
		}

		return nil
	}

	if err := c.statusUpdate(s, "Updating"); err != nil {
		return err
	}

	return nil
}

func (c *StackController) statusUpdate(s *ct.Stack, status string) error {
	cc, err := c.Provider.convoxClient()
	if err != nil {
		return err
	}

	ss, err := cc.ConvoxV1().Stacks(s.ObjectMeta.Namespace).Get(s.ObjectMeta.Name, am.GetOptions{})
	if err != nil {
		return err
	}

	ss.Status = ct.StackStatus(status)

	if _, err := cc.ConvoxV1().Stacks(ss.ObjectMeta.Namespace).Update(ss); err != nil {
		return err
	}

	return nil
}

func assertStack(v interface{}) (*ct.Stack, error) {
	s, ok := v.(*ct.Stack)
	if !ok {
		return nil, fmt.Errorf("could not assert stack for type: %T", v)
	}

	return s, nil
}
