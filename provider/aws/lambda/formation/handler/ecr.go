package handler

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecr"
)

func HandleECRRepository(req Request) (string, map[string]string, error) {
	switch req.RequestType {
	case "Create":
		fmt.Println("CREATING ECRREPOSITORY")
		fmt.Printf("req %+v\n", req)
		return ECRRepositoryCreate(req)
	case "Update":
		fmt.Println("UPDATING ECRREPOSITORY")
		fmt.Printf("req %+v\n", req)
		return ECRRepositoryUpdate(req)
	case "Delete":
		fmt.Println("DELETING ECRREPOSITORY")
		fmt.Printf("req %+v\n", req)
		return ECRRepositoryDelete(req)
	}

	return "", nil, fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

func ECRRepositoryCreate(req Request) (string, map[string]string, error) {
	res, err := ECR(req).CreateRepository(&ecr.CreateRepositoryInput{
		RepositoryName: aws.String(fmt.Sprintf("%s-%s", req.ResourceProperties["RepositoryName"].(string), strings.ToLower(generateId("", 10)))),
	})

	if err != nil {
		return "", nil, err
	}

	outputs := map[string]string{
		"RegistryId":     *res.Repository.RegistryId,
		"RepositoryName": *res.Repository.RepositoryName,
	}

	return *res.Repository.RepositoryArn, outputs, nil
}

func ECRRepositoryUpdate(req Request) (string, map[string]string, error) {
	reg := strings.Split(req.PhysicalResourceId, ":")[4]
	repo := strings.Split(req.PhysicalResourceId, "/")[1]

	outputs := map[string]string{
		"RegistryId":     reg,
		"RepositoryName": repo,
	}

	return req.PhysicalResourceId, outputs, nil
}

func ECRRepositoryDelete(req Request) (string, map[string]string, error) {
	parts := strings.SplitN(req.PhysicalResourceId, "/", 2)

	fmt.Printf("parts %+v\n", parts)

	if len(parts) != 2 {
		fmt.Printf("could not split ecr arn\n")
		return req.PhysicalResourceId, nil, nil
	}

	_, err := ECR(req).DeleteRepository(&ecr.DeleteRepositoryInput{
		RepositoryName: aws.String(parts[1]),
	})

	fmt.Printf("err %+v\n", err)

	// TODO let the cloudformation finish thinking this deleted
	// but take note so we can figure out why
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return req.PhysicalResourceId, nil, nil
	}

	return req.PhysicalResourceId, nil, nil
}
