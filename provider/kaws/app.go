package kaws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/convox/rack/pkg/structs"

	am "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *Provider) AppCreate(name string, opts structs.AppCreateOptions) (*structs.App, error) {
	a, err := p.Provider.AppCreate(name, opts)
	if err != nil {
		return nil, err
	}

	res, err := p.ECR.CreateRepository(&ecr.CreateRepositoryInput{
		RepositoryName: aws.String(fmt.Sprintf("%s/%s", p.Rack, name)),
	})
	if err != nil {
		return nil, err
	}

	ns, err := p.Provider.Cluster.CoreV1().Namespaces().Get(fmt.Sprintf("%s-%s", p.Rack, name), am.GetOptions{})
	if err != nil {
		return nil, err
	}

	if ns.ObjectMeta.Annotations == nil {
		ns.ObjectMeta.Annotations = map[string]string{}
	}

	ns.ObjectMeta.Annotations["convox.registry"] = *res.Repository.RepositoryUri

	if _, err := p.Provider.Cluster.CoreV1().Namespaces().Update(ns); err != nil {
		return nil, err
	}

	return a, nil
}

func (p *Provider) AppDelete(name string) error {
	_, err := p.ECR.DeleteRepository(&ecr.DeleteRepositoryInput{
		Force:          aws.Bool(true),
		RepositoryName: aws.String(fmt.Sprintf("%s/%s", p.Rack, name)),
	})
	if err != nil {
		return err
	}

	return p.Provider.AppDelete(name)
}

func (p *Provider) AppIdles(name string) (bool, error) {
	return false, nil
}

func (p *Provider) AppStatus(name string) (string, error) {
	return "running", nil
}
