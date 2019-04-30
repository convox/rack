package aws

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
	"github.com/fatih/color"
	"golang.org/x/crypto/nacl/secretbox"
	"gopkg.in/cheggaaa/pb.v1"

	cv "github.com/convox/version"
)

const (
	keyLength   = 32
	nonceLength = 24
)

type envelope struct {
	Ciphertext   []byte `json:"c"`
	EncryptedKey []byte `json:"k"`
	Nonce        []byte `json:"n"`
}

func (p *Provider) SystemDecrypt(data []byte) ([]byte, error) {
	var e *envelope

	err := json.Unmarshal(data, &e)
	if err != nil {
		return nil, err
	}

	if len(e.EncryptedKey) == 0 {
		return nil, fmt.Errorf("invalid ciphertext")
	}

	res, err := p.kms().Decrypt(&kms.DecryptInput{
		CiphertextBlob: e.EncryptedKey,
	})
	if err != nil {
		return nil, err
	}

	var key [keyLength]byte
	copy(key[:], res.Plaintext[0:keyLength])

	var nonce [nonceLength]byte
	copy(nonce[:], e.Nonce[0:nonceLength])

	var dec []byte

	dec, ok := secretbox.Open(dec, e.Ciphertext, &nonce, &key)
	if !ok {
		return nil, fmt.Errorf("failed decryption")
	}

	return dec, nil
}

func (p *Provider) SystemEncrypt(data []byte) ([]byte, error) {
	req := &kms.GenerateDataKeyInput{
		KeyId:         aws.String(p.EncryptionKey),
		NumberOfBytes: aws.Int64(keyLength),
	}

	res, err := p.kms().GenerateDataKey(req)
	if err != nil {
		return nil, err
	}

	var key [keyLength]byte
	copy(key[:], res.Plaintext[0:keyLength])

	nres, err := p.kms().GenerateRandom(&kms.GenerateRandomInput{
		NumberOfBytes: aws.Int64(nonceLength),
	})
	if err != nil {
		return nil, err
	}

	var nonce [nonceLength]byte
	copy(nonce[:], nres.Plaintext[0:nonceLength])

	var enc []byte

	enc = secretbox.Seal(enc, data, &nonce, &key)

	e := &envelope{
		Ciphertext:   enc,
		EncryptedKey: res.CiphertextBlob,
		Nonce:        nonce[:],
	}

	return json.Marshal(e)
}

func (p *Provider) SystemGet() (*structs.System, error) {
	log := Logger.At("SystemGet").Start()

	stacks, err := p.describeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(p.Rack),
	})
	if ae, ok := err.(awserr.Error); ok && ae.Code() == "ValidationError" {
		return nil, log.Error(errorNotFound(fmt.Sprintf("%s not found", p.Rack)))
	}
	if err != nil {
		return nil, log.Error(err)
	}
	if len(stacks) != 1 {
		return nil, log.Errorf("could not load stack for app: %s", p.Rack)
	}

	stack := stacks[0]
	status := humanStatus(*stack.StackStatus)
	params := stackParameters(stack)

	count, err := strconv.Atoi(params["InstanceCount"])
	if err != nil {
		return nil, log.Error(err)
	}

	// status precedence: (all other stack statues) > converging > running
	// check if the autoscale group is shuffling instances
	if status == "running" {
		srs, err := p.listStackResources(p.Rack)
		if err != nil {
			return nil, log.Error(err)
		}

		var asgName string
		for _, sr := range srs {
			if *sr.LogicalResourceId == "Instances" {
				asgName = *sr.PhysicalResourceId
				break
			}
		}

		asgres, err := p.autoscaling().DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{
				aws.String(asgName),
			},
		})
		if err != nil {
			return nil, log.Error(err)
		}

		if len(asgres.AutoScalingGroups) <= 0 {
			return nil, log.Errorf("scaling group %s was not found", asgName)
		}

		for _, instance := range asgres.AutoScalingGroups[0].Instances {
			if *instance.LifecycleState != "InService" {
				status = "converging"
				break
			}
		}
	}

	outputs := map[string]string{}

	for _, out := range stack.Outputs {
		outputs[*out.OutputKey] = *out.OutputValue
	}

	version := params["Version"]

	if p.Development {
		version = "dev"
	}

	r := &structs.System{
		Count:      count,
		Domain:     outputs["Domain"],
		Name:       p.Rack,
		Outputs:    outputs,
		Parameters: params,
		Provider:   "aws",
		Region:     p.Region,
		Status:     status,
		Type:       params["InstanceType"],
		Version:    version,
	}

	log.Success()
	return r, nil
}

func (p *Provider) SystemInstall(w io.Writer, opts structs.SystemInstallOptions) (string, error) {
	name := cs(opts.Name, "convox")

	var version string

	if opts.Version != nil {
		version = *opts.Version
	} else {
		v, err := cv.Latest()
		if err != nil {
			return "", err
		}
		version = v
	}

	if err := helpers.AwsCredentialsLoad(); err != nil {
		return "", err
	}

	template := fmt.Sprintf("https://convox.s3.amazonaws.com/release/%s/rack.json", version)

	password := randomString(30)

	params := map[string]string{
		"Password": password,
		"Version":  version,
	}

	if opts.Id != nil {
		params["ClientId"] = *opts.Id
	}

	if opts.Parameters != nil {
		for k, v := range opts.Parameters {
			params[k] = v
		}
	}

	tags := map[string]string{
		"System": "convox",
		"Type":   "rack",
	}

	raw := helpers.DefaultBool(opts.Raw, false)

	var bar *pb.ProgressBar

	cf := cloudformation.New(session.New(&aws.Config{}))

	if !raw {
		fmt.Fprintf(w, "Preparing... ")
	}

	err := helpers.CloudformationInstall(cf, name, template, params, tags, func(current, total int) {
		if raw {
			fmt.Fprintf(w, "{ \"stack\": %q, \"current\": %d, \"total\": %d }\n", name, current, total)
			return
		}

		if bar == nil {
			fmt.Fprintf(w, "OK\n")
			bar = pb.New(total)
			bar.Format(" ██  ")
			bar.Output = w
			bar.Prefix("Installing...")
			bar.ShowBar = false
			bar.ShowCounters = false
			bar.ShowTimeLeft = false
			bar.Start()
		}

		bar.Set(current)
	})
	if err != nil {
		return "", err
	}

	if !raw {
		bar.Finish()
	}

	dres, err := cf.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(name),
	})
	if err != nil {
		return "", err
	}
	if len(dres.Stacks) < 1 {
		return "", fmt.Errorf("could not find stack: %s", name)
	}

	outputs := map[string]string{}

	for _, o := range dres.Stacks[0].Outputs {
		outputs[*o.OutputKey] = *o.OutputValue
	}

	ep := fmt.Sprintf("https://convox:%s@%s", url.QueryEscape(password), outputs["Dashboard"])

	if !raw {
		fmt.Fprintf(w, "Starting... ")
	}

	if err := waitForAvailability(ep); err != nil {
		return "", err
	}

	if !raw {
		fmt.Fprintf(w, "OK, %s\n", outputs["Dashboard"])
	}

	return ep, nil
}

// SystemLogs streams logs for the Rack
func (p *Provider) SystemLogs(opts structs.LogsOptions) (io.ReadCloser, error) {
	group, err := p.rackResource("LogGroup")
	if err != nil {
		return nil, err
	}

	return p.subscribeLogs(group, opts)
}

func (p *Provider) SystemMetrics(opts structs.MetricsOptions) (structs.Metrics, error) {
	metrics := map[string]bool{}

	if opts.Metrics != nil {
		for _, m := range opts.Metrics {
			metrics[m] = true
		}
	}

	ms := structs.Metrics{}

	for _, md := range p.systemMetricDefinitions() {
		if len(metrics) > 0 && !metrics[md.Name] {
			continue
		}

		m, err := p.cloudwatchMetric(md, opts)
		if err != nil {
			return nil, err
		}

		ms = append(ms, *m)
	}

	return ms, nil
}

func (p *Provider) SystemProcesses(opts structs.SystemProcessesOptions) (structs.Processes, error) {
	var tasks []string
	var err error

	if opts.All != nil && *opts.All {
		err := p.ecs().ListTasksPages(&ecs.ListTasksInput{
			Cluster: aws.String(p.Cluster),
		}, func(page *ecs.ListTasksOutput, lastPage bool) bool {
			for _, arn := range page.TaskArns {
				tasks = append(tasks, *arn)
			}
			return true
		})
		if err != nil {
			return nil, err
		}
	} else {
		tasks, err = p.stackTasks(p.Rack)
		if err != nil {
			return nil, err
		}
	}

	ps, err := p.taskProcesses(tasks)
	if err != nil {
		return nil, err
	}

	for i := range ps {
		if ps[i].App == "" {
			ps[i].App = p.Rack
		}
	}

	return ps, nil
}

// SystemReleases lists the latest releases of the rack
func (p *Provider) SystemReleases() (structs.Releases, error) {
	req := &dynamodb.QueryInput{
		KeyConditions: map[string]*dynamodb.Condition{
			"app": {
				AttributeValueList: []*dynamodb.AttributeValue{
					{S: aws.String(p.Rack)},
				},
				ComparisonOperator: aws.String("EQ"),
			},
		},
		IndexName:        aws.String("app.created"),
		Limit:            aws.Int64(20),
		ScanIndexForward: aws.Bool(false),
		TableName:        aws.String(p.DynamoReleases),
	}

	res, err := p.dynamodb().Query(req)
	if err != nil {
		return nil, err
	}

	releases := make(structs.Releases, len(res.Items))

	for i, item := range res.Items {
		r, err := releaseFromItem(item)
		if err != nil {
			return nil, err
		}

		releases[i] = *r
	}

	return releases, nil
}

func (p *Provider) SystemUninstall(name string, w io.Writer, opts structs.SystemUninstallOptions) error {
	if err := helpers.AwsCredentialsLoad(); err != nil {
		return err
	}

	cf := cloudformation.New(session.New(&aws.Config{}))

	dres, err := cf.DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String(name)})
	if err != nil {
		return err
	}
	if len(dres.Stacks) < 1 {
		return fmt.Errorf("could not find rack: %s", name)
	}

	deps, err := rackDependencies(name)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "The following stacks will be deleted:\n")

	for _, d := range deps {
		fmt.Fprintf(w, "  %s\n", d)
	}

	if opts.Input != nil && !cb(opts.Force, false) {
		fmt.Fprintf(w, "Delete everything? [y/N]: ")

		answer, err := bufio.NewReader(opts.Input).ReadString('\n')
		if err != nil {
			return err
		}

		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			return fmt.Errorf("aborting")
		}
	}

	for _, d := range deps {
		tres, err := cf.GetTemplate(&cloudformation.GetTemplateInput{StackName: aws.String(d)})
		if err != nil {
			return err
		}

		fmt.Fprintf(w, color.HiBlueString("Deleting stack: %s\n"), d)

		token := randomString(20)

		_, err = cf.DeleteStack(&cloudformation.DeleteStackInput{
			ClientRequestToken: aws.String(token),
			StackName:          aws.String(d),
		})
		if err != nil {
			return err
		}

		if err := cloudformationProgress(d, token, []byte(*tres.TemplateBody), w); err != nil {
			return err
		}
	}

	return nil
}

func (p *Provider) SystemUpdate(opts structs.SystemUpdateOptions) error {
	changes := map[string]string{}

	// carry forward values from original custom topic resources
	params := map[string]string{
		"AvailabilityZones": p.AvailabilityZones,
		"KmsKey":            p.CustomEncryptionKey,
		"SshKey":            p.SshKey,
	}

	if opts.Parameters != nil {
		for k, v := range opts.Parameters {
			params[k] = v
		}
	}

	if opts.Count != nil {
		params["InstanceCount"] = strconv.Itoa(*opts.Count)
		changes["count"] = strconv.Itoa(*opts.Count)
	}

	if opts.Type != nil {
		params["InstanceType"] = *opts.Type
		changes["type"] = *opts.Type
	}

	var template []byte

	if opts.Version != nil {
		if *opts.Version == "dev" {
			data, err := ioutil.ReadFile("provider/aws/formation/rack.json")
			if err != nil {
				return err
			}

			template = data
		} else {
			res, err := http.Get(fmt.Sprintf("https://convox.s3.amazonaws.com/release/%s/rack.json", *opts.Version))
			if err != nil {
				return err
			}
			defer res.Body.Close()

			data, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return err
			}

			template = data

			params["Version"] = *opts.Version
		}

		changes["version"] = *opts.Version
	}

	// if there is a version update then record it
	if v, ok := changes["version"]; ok {
		_, err := p.dynamodb().PutItem(&dynamodb.PutItemInput{
			Item: map[string]*dynamodb.AttributeValue{
				"id":      {S: aws.String(v)},
				"app":     {S: aws.String(p.Rack)},
				"created": {S: aws.String(p.createdTime())},
			},
			TableName: aws.String(p.DynamoReleases),
		})
		if err != nil {
			return err
		}
	}

	tags := map[string]string{
		"System": "convox",
		"Type":   "rack",
	}

	if err := p.updateStack(p.Rack, template, params, tags, ""); err != nil {
		return err
	}

	// notify about the update
	p.EventSend("rack:update", structs.EventSendOptions{Data: changes})

	return nil
}

func cloudformationProgress(stack, token string, template []byte, w io.Writer) error {
	var formation struct {
		Resources map[string]interface{}
	}

	if err := json.Unmarshal(template, &formation); err != nil {
		return err
	}

	longest := 0

	for k := range formation.Resources {
		if l := len(k); l > longest {
			longest = l
		}
	}

	cf := cloudformation.New(session.New(&aws.Config{}))

	if w == nil {
		w = ioutil.Discard
	}

	events := map[string]cloudformation.StackEvent{}

	for {
		eres, err := cf.DescribeStackEvents(&cloudformation.DescribeStackEventsInput{
			StackName: aws.String(stack),
		})
		if err != nil {
			return nil // stack is gone, we're done
		}

		sort.Slice(eres.StackEvents, func(i, j int) bool { return eres.StackEvents[i].Timestamp.Before(*eres.StackEvents[j].Timestamp) })

		for _, e := range eres.StackEvents {
			if e.ClientRequestToken == nil || *e.ClientRequestToken != token {
				continue
			}

			if _, ok := events[*e.EventId]; !ok {
				line := fmt.Sprintf(fmt.Sprintf("%%-20s  %%-%ds  %%s", longest), *e.ResourceStatus, *e.LogicalResourceId, *e.ResourceType)

				switch *e.ResourceStatus {
				case "CREATE_IN_PROGRESS":
					fmt.Fprintf(w, "%s\n", color.YellowString(line))
				case "CREATE_COMPLETE":
					fmt.Fprintf(w, "%s\n", color.GreenString(line))
				case "CREATE_FAILED":
					fmt.Fprintf(w, "%s\n  ERROR: %s\n", color.RedString(line), *e.ResourceStatusReason)
				case "DELETE_IN_PROGRESS", "DELETE_COMPLETE", "ROLLBACK_IN_PROGRESS", "ROLLBACK_COMPLETE":
					fmt.Fprintf(w, "%s\n", color.RedString(line))
				default:
					fmt.Fprintf(w, "%s\n", line)
				}

				events[*e.EventId] = *e
			}
		}

		dres, err := cf.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: aws.String(stack),
		})
		if awsError(err) == "ValidationError" {
			return nil // stack is gone
		}
		if err != nil {
			return err
		}
		if len(dres.Stacks) < 1 {
			return fmt.Errorf("could not find stack: %s", stack)
		}

		stack := dres.Stacks[0]

		switch *stack.StackStatus {
		case "CREATE_COMPLETE":
			return nil
		case "ROLLBACK_COMPLETE":
			return fmt.Errorf("installation failed")
		}

		time.Sleep(2 * time.Second)
	}
}

func rackDependencies(name string) ([]string, error) {
	cf := cloudformation.New(session.New(&aws.Config{}))

	stacks := []string{}

	req := &cloudformation.DescribeStacksInput{}

	for {
		res, err := cf.DescribeStacks(req)
		if err != nil {
			return nil, err
		}

		for _, s := range res.Stacks {
			// pass on nested stacks
			if s.ParentId != nil {
				continue
			}

			tags := map[string]string{}

			for _, t := range s.Tags {
				tags[*t.Key] = *t.Value
			}

			if tags["Rack"] == name {
				stacks = append(stacks, *s.StackName)
			}
		}

		if res.NextToken == nil {
			break
		}

		req.NextToken = res.NextToken
	}

	stacks = append(stacks, name)

	return stacks, nil
}

func waitForAvailability(url string) error {
	tick := time.Tick(5 * time.Second)
	timeout := time.After(20 * time.Minute)

	c := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	for {
		select {
		case <-tick:
			if _, err := c.Get(url); err == nil {
				return nil
			}
		case <-timeout:
			return fmt.Errorf("timeout")
		}
	}
}
