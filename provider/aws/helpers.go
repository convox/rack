package aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/convox/rack/api/cache"
	"github.com/convox/rack/api/structs"
)

type Template struct {
	Parameters map[string]TemplateParameter
}

type TemplateParameter struct {
	Default     string
	Description string
	NoEcho      bool
	Type        string
}

func awsError(err error) string {
	if ae, ok := err.(awserr.Error); ok {
		return ae.Code()
	}

	return ""
}

func camelize(dasherized string) string {
	tokens := strings.Split(dasherized, "-")

	for i, token := range tokens {
		switch token {
		case "az":
			tokens[i] = "AZ"
		default:
			tokens[i] = strings.Title(token)
		}
	}

	return strings.Join(tokens, "")
}

func cfParams(source map[string]string) map[string]string {
	params := make(map[string]string)

	for key, value := range source {
		var val string
		switch value {
		case "":
			val = "false"
		case "true":
			val = "true"
		default:
			val = value
		}
		params[camelize(key)] = val
	}

	return params
}

func coalesce(s *dynamodb.AttributeValue, def string) string {
	if s != nil {
		return *s.S
	} else {
		return def
	}
}

func buildTemplate(name, section string, data interface{}) (string, error) {
	d, err := Asset(fmt.Sprintf("templates/%s.tmpl", name))
	if err != nil {
		return "", err
	}

	tmpl, err := template.New(section).Funcs(templateHelpers()).Parse(string(d))
	if err != nil {
		return "", err
	}

	var formation bytes.Buffer

	err = tmpl.Execute(&formation, data)
	if err != nil {
		return "", err
	}

	return formation.String(), nil
}

func (p *AWSProvider) createdTime() string {
	if p.IsTest() {
		return time.Time{}.Format(sortableTime)
	}

	return time.Now().Format(sortableTime)
}

func formationParameters(template string) (map[string]bool, error) {
	res, err := http.Get(template)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	formation, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var t Template

	err = json.Unmarshal(formation, &t)

	if err != nil {
		return nil, err
	}

	params := map[string]bool{}

	for key := range t.Parameters {
		params[key] = true
	}

	return params, nil
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

func lastline(data []byte) string {
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	return lines[len(lines)-1]
}

func stackName(app *structs.App) string {
	if _, ok := app.Tags["Rack"]; ok {
		return fmt.Sprintf("%s-%s", app.Tags["Rack"], app.Name)
	}

	return app.Name
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

func templateHelpers() template.FuncMap {
	return template.FuncMap{
		"env": func(s string) string {
			return os.Getenv(s)
		},
		"upper": func(s string) string {
			return upperName(s)
		},
		"value": func(s string) template.HTML {
			return template.HTML(fmt.Sprintf("%q", s))
		},
	}
}

func dashName(name string) string {
	// Myapp -> myapp; MyApp -> my-app
	re := regexp.MustCompile("([a-z])([A-Z])") // lower case letter followed by upper case

	k := re.ReplaceAllString(name, "${1}-${2}")
	return strings.ToLower(k)
}

func upperName(name string) string {
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

/****************************************************************************
 * AWS API HELPERS
 ****************************************************************************/

func (p *AWSProvider) dynamoBatchDeleteItems(wrs []*dynamodb.WriteRequest, tableName string) error {

	if len(wrs) > 0 {

		if len(wrs) <= 25 {
			_, err := p.dynamodb().BatchWriteItem(&dynamodb.BatchWriteItemInput{
				RequestItems: map[string][]*dynamodb.WriteRequest{
					tableName: wrs,
				},
			})
			if err != nil {
				return err
			}

		} else {

			// if more than 25 items to delete, we have to make multiple calls
			maxLen := 25
			for i := 0; i < len(wrs); i += maxLen {
				high := i + maxLen
				if high > len(wrs) {
					high = len(wrs)
				}

				_, err := p.dynamodb().BatchWriteItem(&dynamodb.BatchWriteItemInput{
					RequestItems: map[string][]*dynamodb.WriteRequest{
						tableName: wrs[i:high],
					},
				})
				if err != nil {
					return err
				}

			}
		}
	} else {
		fmt.Println("ns=api fn=dynamoBatchDeleteItems level=info msg=\"no builds to delete\"")
	}

	return nil
}

func (p *AWSProvider) describeStacks(input *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
	res, ok := cache.Get("describeStacks", input.StackName).(*cloudformation.DescribeStacksOutput)

	if ok {
		return res, nil
	}

	res, err := p.cloudformation().DescribeStacks(input)

	if err != nil {
		return nil, err
	}

	if !p.SkipCache {
		if err := cache.Set("describeStacks", input.StackName, res, 5*time.Second); err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (p *AWSProvider) describeStack(name string) (*cloudformation.Stack, error) {
	res, err := p.describeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(name),
	})
	if ae, ok := err.(awserr.Error); ok && ae.Code() == "ValidationError" {
		return nil, ErrorNotFound(fmt.Sprintf("%s not found", name))
	}
	if err != nil {
		return nil, err
	}
	if len(res.Stacks) != 1 {
		return nil, fmt.Errorf("could not load stack: %s", name)
	}

	return res.Stacks[0], nil
}

func (p *AWSProvider) describeStackEvents(input *cloudformation.DescribeStackEventsInput) (*cloudformation.DescribeStackEventsOutput, error) {
	res, ok := cache.Get("describeStackEvents", input.StackName).(*cloudformation.DescribeStackEventsOutput)

	if ok {
		return res, nil
	}

	res, err := p.cloudformation().DescribeStackEvents(input)
	if err != nil {
		return nil, err
	}

	if !p.SkipCache {
		if err := cache.Set("describeStackEvents", input.StackName, res, 5*time.Second); err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (p *AWSProvider) describeTaskDefinition(name string) (*ecs.TaskDefinition, error) {
	td, ok := cache.Get("describeTaskDefinition", name).(*ecs.TaskDefinition)
	if ok {
		return td, nil
	}

	res, err := p.ecs().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(name),
	})
	if ae, ok := err.(awserr.Error); ok && ae.Code() == "ValidationError" {
		return nil, ErrorNotFound(fmt.Sprintf("%s not found", name))
	}
	if err != nil {
		return nil, err
	}

	td = res.TaskDefinition

	if !p.SkipCache {
		if err := cache.Set("describeTaskDefinition", name, td, 10*time.Second); err != nil {
			return nil, err
		}
	}

	return td, nil
}

func (p *AWSProvider) listContainerInstances(input *ecs.ListContainerInstancesInput) (*ecs.ListContainerInstancesOutput, error) {
	res, ok := cache.Get("listContainerInstances", input).(*ecs.ListContainerInstancesOutput)

	if ok {
		return res, nil
	}

	res, err := p.ecs().ListContainerInstances(input)

	if err != nil {
		return nil, err
	}

	if !p.SkipCache {
		if err := cache.Set("listContainerInstances", input, res, 10*time.Second); err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (p *AWSProvider) s3Exists(bucket, key string) (bool, error) {
	_, err := p.s3().HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		if aerr, ok := err.(awserr.RequestFailure); ok && aerr.StatusCode() == 404 {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (p *AWSProvider) s3Get(bucket, key string) ([]byte, error) {
	req := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	res, err := p.s3().GetObject(req)

	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(res.Body)
}

func (p *AWSProvider) s3Delete(bucket, key string) error {
	req := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err := p.s3().DeleteObject(req)

	return err
}

func (p *AWSProvider) s3Put(bucket, key string, data []byte, public bool) error {
	req := &s3.PutObjectInput{
		Body:          bytes.NewReader(data),
		Bucket:        aws.String(bucket),
		ContentLength: aws.Int64(int64(len(data))),
		Key:           aws.String(key),
	}

	if public {
		req.ACL = aws.String("public-read")
	}

	_, err := p.s3().PutObject(req)

	return err
}

// updateStack updates a stack
//   template is url to a template or empty string to reuse previous
//   changes is a list of parameter changes to make (does not need to include every param)
func (p *AWSProvider) updateStack(name string, template string, changes map[string]string) error {
	cache.Clear("describeStacks", nil)
	cache.Clear("describeStacks", name)

	req := &cloudformation.UpdateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(name),
	}

	params := map[string]bool{}

	if template != "" {
		req.TemplateURL = aws.String(template)

		fp, err := formationParameters(template)
		if err != nil {
			return err
		}

		for p := range fp {
			params[p] = true
		}
	} else {
		req.UsePreviousTemplate = aws.Bool(true)

		stack, err := p.describeStack(name)
		if err != nil {
			return err
		}

		for _, p := range stack.Parameters {
			params[*p.ParameterKey] = true
		}
	}

	sorted := []string{}

	for param := range params {
		sorted = append(sorted, param)
	}

	// sort params for easier testing
	sort.Strings(sorted)

	for _, param := range sorted {
		if value, ok := changes[param]; ok {
			req.Parameters = append(req.Parameters, &cloudformation.Parameter{
				ParameterKey:   aws.String(param),
				ParameterValue: aws.String(value),
			})
		} else {
			req.Parameters = append(req.Parameters, &cloudformation.Parameter{
				ParameterKey:     aws.String(param),
				UsePreviousValue: aws.Bool(true),
			})
		}
	}

	_, err := p.cloudformation().UpdateStack(req)

	cache.Clear("describeStacks", nil)
	cache.Clear("describeStacks", name)

	return err
}

// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-types.html
var instanceTypes = []string{
	"c1.medium",
	"c1.xlarge",
	"c3.2xlarge",
	"c3.4xlarge",
	"c3.8xlarge",
	"c3.large",
	"c3.xlarge",
	"c4.2xlarge",
	"c4.4xlarge",
	"c4.8xlarge",
	"c4.large",
	"c4.xlarge",
	"cc1.4xlarge",
	"cc2.8xlarge",
	"cg1.4xlarge",
	"cr1.8xlarge",
	"d2.2xlarge",
	"d2.4xlarge",
	"d2.8xlarge",
	"d2.xlarge",
	"g2.2xlarge",
	"g2.8xlarge",
	"hi1.4xlarge",
	"hs1.8xlarge",
	"i2.2xlarge",
	"i2.4xlarge",
	"i2.8xlarge",
	"i2.xlarge",
	"m1.large",
	"m1.medium",
	"m1.small",
	"m1.xlarge",
	"m2.2xlarge",
	"m2.4xlarge",
	"m2.xlarge",
	"m3.2xlarge",
	"m3.large",
	"m3.medium",
	"m3.xlarge",
	"m4.10xlarge",
	"m4.2xlarge",
	"m4.4xlarge",
	"m4.large",
	"m4.xlarge",
	"r3.2xlarge",
	"r3.4xlarge",
	"r3.8xlarge",
	"r3.large",
	"r3.xlarge",
	"t1.micro",
	"t2.large",
	"t2.medium",
	"t2.micro",
	"t2.nano",
	"t2.small",
	"x1.16xlarge",
	"x1.32xlarge",
	"x1.4xlarge",
	"x1.8xlarge",
}
