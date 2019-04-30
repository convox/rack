package kaws

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecr"
)

var (
	reECRHost = regexp.MustCompile(`(\d+)\.dkr\.ecr\.([^.]+)\.amazonaws\.com`)
)

func (p *Provider) RepositoryAuth(app string) (string, string, error) {
	repo, _, err := p.RepositoryHost(app)
	if err != nil {
		return "", "", err
	}

	if !reECRHost.MatchString(repo) {
		return "", "", fmt.Errorf("invalid ecr repository: %s", repo)
	}

	registry := reECRHost.FindStringSubmatch(repo)

	res, err := p.ECR.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{
		RegistryIds: []*string{aws.String(registry[1])},
	})
	if err != nil {
		return "", "", err
	}
	if len(res.AuthorizationData) != 1 {
		return "", "", fmt.Errorf("no authorization data")
	}

	token, err := base64.StdEncoding.DecodeString(*res.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		return "", "", err
	}

	parts := strings.SplitN(string(token), ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid auth data")
	}

	return parts[0], parts[1], nil
}

func (p *Provider) RepositoryHost(app string) (string, bool, error) {
	registry, err := p.appRegistry(app)
	if err != nil {
		return "", false, err
	}

	return registry, true, nil
}
