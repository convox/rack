package models

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

func (s *Service) CreateDatastore() (*cloudformation.CreateStackInput, error) {
	formation, err := s.Formation()
	if err != nil {
		return nil, err
	}

	if s.Type != "redis" {
		s.Parameters["Password"] = generateId("", 30)
	}

	// SubnetsPrivate is a List<AWS::EC2::Subnet::Id> and can not be empty
	// So reuse SUBNETS if SUBNETS_PRIVATE is not set
	subnetsPrivate := os.Getenv("SUBNETS_PRIVATE")
	if subnetsPrivate == "" {
		subnetsPrivate = os.Getenv("SUBNETS")
	}

	s.Parameters["Subnets"] = os.Getenv("SUBNETS")
	s.Parameters["SubnetsPrivate"] = subnetsPrivate
	s.Parameters["Vpc"] = os.Getenv("VPC")
	s.Parameters["VpcCidr"] = os.Getenv("VPCCIDR")

	req := &cloudformation.CreateStackInput{
		StackName:    aws.String(s.StackName()),
		TemplateBody: aws.String(formation),
	}

	return req, nil
}

func (s *Service) UpdateDatastore() (*cloudformation.UpdateStackInput, error) {
	formation, err := s.Formation()
	if err != nil {
		return nil, err
	}

	// Special case new parameters that don't have a default value

	// SubnetsPrivate is a List<AWS::EC2::Subnet::Id> and can not be empty
	// So reuse SUBNETS if SUBNETS_PRIVATE is not set	subnetsPrivate := os.Getenv("SUBNETS_PRIVATE")
	subnetsPrivate := os.Getenv("SUBNETS_PRIVATE")
	if subnetsPrivate == "" {
		subnetsPrivate = os.Getenv("SUBNETS")
	}

	s.Parameters["Subnets"] = os.Getenv("SUBNETS")
	s.Parameters["SubnetsPrivate"] = subnetsPrivate
	s.Parameters["Vpc"] = os.Getenv("VPC")
	s.Parameters["VpcCidr"] = os.Getenv("VPCCIDR")

	req := &cloudformation.UpdateStackInput{
		StackName:    aws.String(s.StackName()),
		TemplateBody: aws.String(formation),
	}

	return req, nil
}
