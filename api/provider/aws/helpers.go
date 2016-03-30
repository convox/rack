package aws

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/dynamodb"
)

type Template struct {
	Parameters map[string]TemplateParameter
}

type TemplateParameter struct {
	Default     string
	Description string
	Type        string
}

func awsError(err error) string {
	if ae, ok := err.(awserr.Error); ok {
		return ae.Code()
	}

	return ""
}

func coalesce(s *dynamodb.AttributeValue, def string) string {
	if s != nil {
		return *s.S
	} else {
		return def
	}
}

func formationParameters(formation string) (map[string]TemplateParameter, error) {
	var t Template

	err := json.Unmarshal([]byte(formation), &t)

	if err != nil {
		return nil, err
	}

	return t.Parameters, nil
}

func humanStatus(original string) string {
	switch original {
	case "":
		return "new"
	case "CREATE_IN_PROGRESS":
		return "creating"
	case "CREATE_COMPLETE":
		return "running"
	case "DELETE_FAILED":
		return "running"
	case "DELETE_IN_PROGRESS":
		return "deleting"
	case "ROLLBACK_IN_PROGRESS":
		return "rollback"
	case "ROLLBACK_COMPLETE":
		return "failed"
	case "UPDATE_IN_PROGRESS":
		return "updating"
	case "UPDATE_COMPLETE_CLEANUP_IN_PROGRESS":
		return "updating"
	case "UPDATE_COMPLETE":
		return "running"
	case "UPDATE_ROLLBACK_IN_PROGRESS":
		return "rollback"
	case "UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS":
		return "rollback"
	case "UPDATE_ROLLBACK_COMPLETE":
		return "running"
	case "UPDATE_ROLLBACK_FAILED":
		return "running"
	default:
		fmt.Printf("unknown status: %s\n", original)
		return "unknown"
	}
}

func stackParameters(stack *cloudformation.Stack) map[string]string {
	parameters := make(map[string]string)

	for _, parameter := range stack.Parameters {
		parameters[*parameter.ParameterKey] = *parameter.ParameterValue
	}

	return parameters
}

func stackOutputs(stack *cloudformation.Stack) map[string]string {
	outputs := make(map[string]string)

	for _, output := range stack.Outputs {
		outputs[*output.OutputKey] = *output.OutputValue
	}

	return outputs
}

func stackTags(stack *cloudformation.Stack) map[string]string {
	tags := make(map[string]string)

	for _, tag := range stack.Tags {
		tags[*tag.Key] = *tag.Value
	}

	return tags
}

func (p *AWSProvider) stackUpdate(name string, templateUrl string, changes map[string]string) error {
	app, err := p.AppGet(name)

	if err != nil {
		return err
	}

	params := map[string]string{}

	for key, value := range app.Parameters {
		params[key] = value
	}

	for key, value := range changes {
		params[key] = value
	}

	req := &cloudformation.UpdateStackInput{
		StackName:    aws.String(name),
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
	}

	if templateUrl != "" {
		res, err := http.Get(templateUrl)

		if err != nil {
			return err
		}

		body, err := ioutil.ReadAll(res.Body)

		if err != nil {
			return err
		}

		fp, err := formationParameters(string(body))

		// remove params that don't exist in the template
		for key := range params {
			if _, ok := fp[key]; !ok {
				delete(params, key)
			}
		}

		req.TemplateURL = aws.String(templateUrl)
	}

	for key, value := range params {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(key),
			ParameterValue: aws.String(value),
		})
	}

	_, err = p.updateStack(req)

	return err
}
