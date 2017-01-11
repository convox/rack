package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/api/structs"
)

// ServiceCreate creates a new resource.
// Note: see also createService() below.
func (p *AWSProvider) ServiceCreate(name, kind string, params map[string]string) (*structs.Resource, error) {
	return p.ResourceCreate(name, kind, params)
}

// ServiceDelete deletes a resource.
func (p *AWSProvider) ServiceDelete(name string) (*structs.Resource, error) {
	return p.ResourceDelete(name)
}

// ServiceGet retrieves a resource.
func (p *AWSProvider) ServiceGet(name string) (*structs.Resource, error) {
	return p.ResourceGet(name)
}

//resourceApps returns the apps that have been linked with a resource (ignoring apps that have been delete out of band)
func (p *AWSProvider) serviceApps(s structs.Resource) (structs.Apps, error) {
	return p.resourceApps(s)
}

// ServiceList lists the resources.
func (p *AWSProvider) ServiceList() (structs.Resources, error) {
	return p.ResourceList()
}

// ServiceLink creates a link between the provided app and resource.
func (p *AWSProvider) ServiceLink(name, app, process string) (*structs.Resource, error) {
	return p.ResourceLink(name, app, process)
}

// ServiceUnlink removes a link between the provided app and resource.
func (p *AWSProvider) ServiceUnlink(name, app, process string) (*structs.Resource, error) {
	return p.ResourceUnlink(name, app, process)
}

// ServiceUpdate updates a resource with new params.
func (p *AWSProvider) ServiceUpdate(name string, params map[string]string) (*structs.Resource, error) {
	return p.ResourceUpdate(name, params)
}

// createService creates a Resource.
// Note: see also ServiceCreate() above.
// This should probably be renamed to createServiceStack to be in conformity with createServiceURL below.
func (p *AWSProvider) createService(s *structs.Resource) (*cloudformation.CreateStackInput, error) {
	return p.createResource(s)
}

func (p *AWSProvider) createServiceURL(s *structs.Resource, allowedProtocols ...string) (*cloudformation.CreateStackInput, error) {
	return p.createResourceURL(s, allowedProtocols...)
}

func (p *AWSProvider) updateService(s *structs.Resource) error {
	return p.updateResource(s)
}

// add to links
func (p *AWSProvider) linkService(a *structs.App, s *structs.Resource) error {
	return p.linkResource(a, s)
	/*
		s.Apps = append(s.Apps, *a)

		return p.updateResource(s)
	*/
}

// delete from links
func (p *AWSProvider) unlinkService(a *structs.App, s *structs.Resource) error {
	return p.unlinkResource(a, s)
	/*
		apps := structs.Apps{}
		for _, linkedApp := range s.Apps {
			if a.Name != linkedApp.Name {
				apps = append(apps, linkedApp)
			}
		}

		s.Apps = apps

		return p.updateResource(s)
	*/
}

/*
func (p *AWSProvider) appendSystemParameters(s *structs.Resource) error {
	password, err := generatePassword()
	if err != nil {
		return err
	}

	s.Parameters["Password"] = password
	s.Parameters["SecurityGroups"] = p.SecurityGroup
	s.Parameters["Subnets"] = p.Subnets
	s.Parameters["SubnetsPrivate"] = coalesceString(p.SubnetsPrivate, p.Subnets)
	s.Parameters["Vpc"] = p.Vpc
	s.Parameters["VpcCidr"] = p.VpcCidr

	return nil
}

func coalesceString(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

func filterFormationParameters(s *structs.Resource, formation string) error {
	var params struct {
		Parameters map[string]interface{}
	}

	if err := json.Unmarshal([]byte(formation), &params); err != nil {
		return err
	}

	for key := range s.Parameters {
		if _, ok := params.Parameters[key]; !ok {
			delete(s.Parameters, key)
		}
	}

	return nil
}

func generatePassword() (string, error) {
	data := make([]byte, 1024)

	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)

	return hex.EncodeToString(hash[:])[0:30], nil
}
*/

func serviceFormation(kind string, data interface{}) (string, error) {
	d, err := buildTemplate(fmt.Sprintf("resource/%s", kind), "resource", data)
	if err != nil {
		return "", err
	}

	return d, nil
}

func serviceFromStack(stack *cloudformation.Stack) structs.Resource {
	params := stackParameters(stack)
	tags := stackTags(stack)
	name := coalesceString(tags["Name"], *stack.StackName)

	exports := map[string]string{}

	if url, ok := params["Url"]; ok {
		exports["URL"] = url
	}

	return structs.Resource{
		Name:       name,
		Stack:      *stack.StackName,
		Type:       tags["Resource"],
		Status:     humanStatus(*stack.StackStatus),
		Outputs:    stackOutputs(stack),
		Parameters: params,
		Tags:       tags,
		Exports:    exports,
	}
}
