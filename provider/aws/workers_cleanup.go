package aws

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/convox/logger"
	"github.com/convox/rack/helpers"
	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
)

func (p *AWSProvider) workerCleanup() {
	log := logger.New("ns=workers.cleanup")

	defer recoverWith(func(err error) {
		helpers.Error(log, err)
	})

	for range time.Tick(1 * time.Hour) {
		p.cleanupBuilds(log)
	}
}

func (p *AWSProvider) cleanupBuilds(log *logger.Logger) error {
	as, err := p.AppList()
	if err != nil {
		return log.Error(err)
	}

	for _, a := range as {
		log = log.Replace("app", a.Name)

		log = log.At("builds")
		if count, err := p.cleanupAppBuilds(a); err != nil {
			log.Error(err)
		} else {
			log.Logf("expired=%d", count)
		}

		log = log.At("images")
		if count, err := p.cleanupAppImages(a); err != nil {
			log.Error(err)
		} else {
			log.Logf("expired=%d", count)
		}
	}

	return nil
}

func (p *AWSProvider) cleanupAppBuilds(a structs.App) (int, error) {
	active, err := p.activeBuild(a)
	if err != nil {
		return 0, err
	}

	removed := 0

	for {
		bs, err := p.BuildList(a.Name, structs.BuildListOptions{Count: options.Int(1000)})
		if err != nil {
			return 0, err
		}

		if len(bs) <= maxBuilds {
			break
		}

		remove := []string{}

		for _, b := range bs[maxBuilds:] {
			if b.Id != active {
				remove = append(remove, b.Id)
			}
		}

		for _, rc := range chunk(remove, 25) {
			req := &dynamodb.BatchWriteItemInput{
				RequestItems: map[string][]*dynamodb.WriteRequest{p.DynamoBuilds: {}},
			}

			for _, id := range rc {
				req.RequestItems[p.DynamoBuilds] = append(req.RequestItems[p.DynamoBuilds], &dynamodb.WriteRequest{
					DeleteRequest: &dynamodb.DeleteRequest{
						Key: map[string]*dynamodb.AttributeValue{
							"id": {S: aws.String(id)},
						},
					},
				})
			}

			if _, err := p.dynamodb().BatchWriteItem(req); err != nil {
				return 0, err
			}

			removed += len(rc)
		}
	}

	return removed, nil
}

func (p *AWSProvider) cleanupAppImages(a structs.App) (int, error) {
	active, err := p.activeBuild(a)
	if err != nil {
		return 0, err
	}

	bs, err := p.BuildList(a.Name, structs.BuildListOptions{Count: options.Int(maxBuilds)})
	if err != nil {
		return 0, err
	}

	if len(bs) < maxBuilds {
		return 0, nil
	}

	bh := map[string]bool{}

	for _, b := range bs {
		bh[b.Id] = true
	}

	repo, err := p.appRepositoryName(a)
	if err != nil {
		return 0, err
	}

	tags, err := p.repoTags(repo)
	if err != nil {
		return 0, err
	}

	remove := []string{}

	for _, tag := range tags {
		parts := strings.SplitN(tag, ".", 2)
		if len(parts) < 2 || !strings.HasPrefix(parts[1], "B") {
			continue
		}

		if _, ok := bh[parts[1]]; !ok && parts[1] != active {
			remove = append(remove, tag)
		}
	}

	if len(remove) == 0 {
		return 0, nil
	}

	for _, rc := range chunk(remove, 100) {
		req := &ecr.BatchDeleteImageInput{
			RepositoryName: aws.String(repo),
		}

		for _, tag := range rc {
			req.ImageIds = append(req.ImageIds, &ecr.ImageIdentifier{
				ImageTag: aws.String(tag),
			})
		}

		if _, err := p.ecr().BatchDeleteImage(req); err != nil {
			return 0, err
		}
	}

	return len(remove), nil
}

func (p *AWSProvider) activeBuild(a structs.App) (string, error) {
	if a.Release == "" {
		return "", nil
	}

	r, err := p.ReleaseGet(a.Name, a.Release)
	if err != nil {
		return "", err
	}

	return r.Build, nil
}

func (p *AWSProvider) repoTags(repo string) ([]string, error) {
	tags := map[string]bool{}

	err := p.ecr().ListImagesPages(&ecr.ListImagesInput{RepositoryName: aws.String(repo)}, func(page *ecr.ListImagesOutput, last bool) bool {
		for _, i := range page.ImageIds {
			if i.ImageTag != nil && *i.ImageTag != "" {
				tags[*i.ImageTag] = true
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	ts := []string{}

	for t := range tags {
		ts = append(ts, t)
	}

	return ts, nil
}

func (p *AWSProvider) appRepositoryName(a structs.App) (string, error) {
	switch a.Generation {
	case "1":
		return a.Outputs["RegistryRepository"], nil
	case "2":
		return p.appResource(a.Name, "Registry")
	}

	return "", fmt.Errorf("unknown generation: %s", a.Generation)
}

func chunk(ss []string, count int) [][]string {
	chunks := [][]string{}

	for {
		if len(ss) <= count {
			return append(chunks, ss)
		}

		chunks = append(chunks, ss[0:count])
		ss = ss[count:]
	}
}
