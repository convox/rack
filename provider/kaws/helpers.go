package kaws

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	cv "github.com/convox/rack/provider/kaws/pkg/client/clientset/versioned"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *Provider) appRegistry(app string) (string, error) {
	ns, err := p.Provider.Cluster.CoreV1().Namespaces().Get(fmt.Sprintf("%s-%s", p.Rack, app), am.GetOptions{})
	if err != nil {
		return "", err
	}

	registry, ok := ns.ObjectMeta.Annotations["convox.registry"]
	if !ok {
		return "", fmt.Errorf("no registry for app: %s", app)
	}

	return registry, nil
}

func (p *Provider) convoxClient() (cv.Interface, error) {
	return cv.NewForConfig(p.Config)
}

func (p *Provider) stackOutputs(stack string) (map[string]string, error) {
	res, err := p.CloudFormation.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stack),
	})
	if err != nil {
		return nil, err
	}
	if len(res.Stacks) != 1 {
		return nil, fmt.Errorf("no such stack: %s", stack)
	}

	outputs := map[string]string{}

	for _, o := range res.Stacks[0].Outputs {
		outputs[*o.OutputKey] = *o.OutputValue
	}

	return outputs, nil
}

func (p *Provider) stackTags(stack string) (map[string]string, error) {
	res, err := p.CloudFormation.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stack),
	})
	if err != nil {
		return nil, err
	}
	if len(res.Stacks) != 1 {
		return nil, fmt.Errorf("no such stack: %s", stack)
	}

	tags := map[string]string{}

	for _, t := range res.Stacks[0].Tags {
		tags[*t.Key] = *t.Value
	}

	return tags, nil
}

func (p *Provider) watchForProcessTermination(ctx context.Context, app, pid string, cancel func()) {
	defer cancel()

	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if _, err := p.ProcessGet(app, pid); err != nil {
				time.Sleep(2 * time.Second)
				cancel()
				return
			}
		}
	}
}

func cloudformationErrorNoUpdates(err error) bool {
	if ae, ok := err.(awserr.Error); ok {
		if ae.Code() == "ValidationError" && strings.Contains(ae.Message(), "No updates are to be performed") {
			return true
		}
	}
	return false
}

func kubectl(args ...string) error {
	cmd := exec.Command("kubectl", args...)

	cmd.Env = os.Environ()

	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(strings.TrimSpace(string(out)))
	}

	return nil
}

var outputConverter = regexp.MustCompile("([a-z])([A-Z])") // lower case letter followed by upper case

func outputToEnvironment(name string) string {
	return strings.ToUpper(outputConverter.ReplaceAllString(name, "${1}_${2}"))
}

func upperName(name string) string {
	if name == "" {
		return ""
	}

	// replace underscores with dashes
	name = strings.Replace(name, "_", "-", -1)

	// myapp -> Myapp; my-app -> MyApp
	us := strings.ToUpper(name[0:1]) + name[1:]

	for {
		i := strings.Index(us, "-")

		if i == -1 {
			break
		}

		s := us[0:i]

		if len(us) > i+1 {
			s += strings.ToUpper(us[i+1 : i+2])
		}

		if len(us) > i+2 {
			s += us[i+2:]
		}

		us = s
	}

	return us
}
