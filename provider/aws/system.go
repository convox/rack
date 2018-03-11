package aws

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/convox/rack/structs"
	"golang.org/x/crypto/nacl/secretbox"
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

func (p *AWSProvider) SystemDecrypt(data []byte) ([]byte, error) {
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

func (p *AWSProvider) SystemEncrypt(data []byte) ([]byte, error) {
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

func (p *AWSProvider) SystemGet() (*structs.System, error) {
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

	// Check if ECS is rescheduling services
	if status == "running" {
		lreq := &ecs.ListServicesInput{
			Cluster:    aws.String(p.Cluster),
			MaxResults: aws.Int64(10),
		}
	Loop:
		for {
			lres, err := p.ecs().ListServices(lreq)
			if err != nil {
				return nil, log.Error(err)
			}

			dres, err := p.describeServices(&ecs.DescribeServicesInput{
				Cluster:  aws.String(p.Cluster),
				Services: lres.ServiceArns,
			})
			if err != nil {
				return nil, log.Error(err)
			}

			for _, s := range dres.Services {
				for _, d := range s.Deployments {
					if *d.RunningCount != *d.DesiredCount {
						status = "converging"
						break Loop
					}
				}
			}

			if lres.NextToken == nil {
				break
			}

			lreq.NextToken = lres.NextToken
		}
	}

	outputs := map[string]string{}

	for _, out := range stack.Outputs {
		outputs[*out.OutputKey] = *out.OutputValue
	}

	r := &structs.System{
		Count:      count,
		Domain:     outputs["Domain"],
		Name:       p.Rack,
		Outputs:    outputs,
		Parameters: params,
		Region:     p.Region,
		Status:     status,
		Type:       params["InstanceType"],
		Version:    params["Version"],
	}

	log.Success()
	return r, nil
}

func (p *AWSProvider) SystemInstall(name string, opts structs.SystemInstallOptions) (string, error) {
	return "", fmt.Errorf("unimplemented")
}

// SystemLogs streams logs for the Rack
func (p *AWSProvider) SystemLogs(opts structs.LogsOptions) (io.ReadCloser, error) {
	group, err := p.rackResource("LogGroup")
	if err != nil {
		return nil, err
	}

	return p.subscribeLogs(group, opts)
}

func (p *AWSProvider) SystemProcesses(opts structs.SystemProcessesOptions) (structs.Processes, error) {
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
func (p *AWSProvider) SystemReleases() (structs.Releases, error) {
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

func (p *AWSProvider) SystemUpdate(opts structs.SystemUpdateOptions) error {
	changes := map[string]string{}
	params := opts.Parameters
	template := ""

	if params == nil {
		params = map[string]string{}
	}

	if opts.InstanceCount != nil {
		params["InstanceCount"] = strconv.Itoa(*opts.InstanceCount)
		changes["count"] = strconv.Itoa(*opts.InstanceCount)
	}

	if opts.InstanceType != nil {
		params["InstanceType"] = *opts.InstanceType
		changes["type"] = *opts.InstanceType
	}

	if opts.Version != nil {
		template = fmt.Sprintf("https://convox.s3.amazonaws.com/release/%s/rack.json", *opts.Version)
		params["Version"] = *opts.Version
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

	if err := p.updateStack(p.Rack, template, params); err != nil {
		return err
	}

	// notify about the update
	p.EventSend("rack:update", structs.EventSendOptions{Data: changes})

	return nil
}
