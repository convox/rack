package aws

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/convox/rack/pkg/structs"
)

const smMaxSecretSize = 65536

var (
	smFailures   = map[string]int{}
	smFailuresMu sync.Mutex
)

func (p *Provider) secretsManagerWrite(rack, app string, env map[string]string) (string, []string, error) {
	data, err := json.Marshal(env)
	if err != nil {
		return "", nil, err
	}

	if len(data) > smMaxSecretSize {
		return "", nil, fmt.Errorf("env data exceeds 64KB Secrets Manager limit (%d bytes)", len(data))
	}

	name := fmt.Sprintf("%s/%s", rack, app)
	val := string(data)

	res, err := p.secretsmanager().CreateSecret(&secretsmanager.CreateSecretInput{
		Name:         aws.String(name),
		SecretString: aws.String(val),
		Tags: []*secretsmanager.Tag{
			{Key: aws.String("convox:rack"), Value: aws.String(rack)},
			{Key: aws.String("convox:app"), Value: aws.String(app)},
		},
	})
	if err != nil {
		ae, ok := err.(awserr.Error)
		if !ok {
			p.smRecordFailure(app)
			return "", nil, err
		}

		switch ae.Code() {
		case secretsmanager.ErrCodeResourceExistsException:
			arn, keys, err := p.secretsManagerUpdate(name, val, env)
			if err != nil {
				p.smRecordFailure(app)
				return "", nil, err
			}
			p.smRecordSuccess(app)
			return arn, keys, nil
		case secretsmanager.ErrCodeInvalidRequestException:
			if restoreErr := p.secretsManagerRestore(name); restoreErr != nil {
				p.smRecordFailure(app)
				return "", nil, fmt.Errorf("restore deleted secret %s: %s", name, restoreErr)
			}
			arn, keys, err := p.secretsManagerUpdate(name, val, env)
			if err != nil {
				p.smRecordFailure(app)
				return "", nil, err
			}
			p.smRecordSuccess(app)
			return arn, keys, nil
		default:
			p.smRecordFailure(app)
			return "", nil, err
		}
	}

	keys := sortedKeys(env)
	p.smRecordSuccess(app)
	return aws.StringValue(res.ARN), keys, nil
}

func (p *Provider) secretsManagerUpdate(name, val string, env map[string]string) (string, []string, error) {
	desc, err := p.secretsmanager().DescribeSecret(&secretsmanager.DescribeSecretInput{
		SecretId: aws.String(name),
	})
	if err != nil {
		return "", nil, err
	}

	_, err = p.secretsManagerPutWithRetry(name, val)
	if err != nil {
		return "", nil, err
	}

	keys := sortedKeys(env)
	return aws.StringValue(desc.ARN), keys, nil
}

func (p *Provider) secretsManagerPutWithRetry(name, val string) (*secretsmanager.PutSecretValueOutput, error) {
	var lastErr error
	for i := 0; i < 3; i++ {
		res, err := p.secretsmanager().PutSecretValue(&secretsmanager.PutSecretValueInput{
			SecretId:     aws.String(name),
			SecretString: aws.String(val),
		})
		if err == nil {
			return res, nil
		}
		lastErr = err
		ae, ok := err.(awserr.Error)
		if !ok || (ae.Code() != "ThrottlingException" && ae.Code() != "InternalServiceError") {
			return nil, err
		}
		time.Sleep(time.Duration(1<<uint(i)) * time.Second)
	}
	return nil, lastErr
}

func (p *Provider) secretsManagerRestore(name string) error {
	_, err := p.secretsmanager().RestoreSecret(&secretsmanager.RestoreSecretInput{
		SecretId: aws.String(name),
	})
	return err
}

func (p *Provider) secretsManagerDelete(rack, app string) error {
	name := fmt.Sprintf("%s/%s", rack, app)

	_, err := p.secretsmanager().DeleteSecret(&secretsmanager.DeleteSecretInput{
		SecretId: aws.String(name),
	})
	if err != nil {
		ae, ok := err.(awserr.Error)
		if ok && ae.Code() == secretsmanager.ErrCodeResourceNotFoundException {
			return nil
		}
		return err
	}

	return nil
}

func (p *Provider) secretsManagerGetARN(rack, app string) (string, error) {
	name := fmt.Sprintf("%s/%s", rack, app)

	res, err := p.secretsmanager().DescribeSecret(&secretsmanager.DescribeSecretInput{
		SecretId: aws.String(name),
	})
	if err != nil {
		return "", err
	}

	return aws.StringValue(res.ARN), nil
}

func (p *Provider) smRecordFailure(app string) {
	var shouldNotify bool
	smFailuresMu.Lock()
	smFailures[app]++
	shouldNotify = smFailures[app] == 3
	smFailuresMu.Unlock()
	if shouldNotify {
		p.EventSend("rack:warning", structs.EventSendOptions{
			Data: map[string]string{
				"message": fmt.Sprintf("Secrets Manager has failed 3 consecutive times for app %s. Consider setting SecretsManagerEnv=No or investigating SM availability.", app),
			},
		})
	}
}

func (p *Provider) smRecordSuccess(app string) {
	smFailuresMu.Lock()
	defer smFailuresMu.Unlock()
	delete(smFailures, app)
}

func sortedKeys(env map[string]string) []string {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
