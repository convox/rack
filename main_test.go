package main

import (
	"bytes"
	"encoding/json"
	"testing"

  yaml "gopkg.in/yaml.v2"
)

type Cases []struct {
	got, want interface{}
}

func TestStaging(t *testing.T) {
	var manifest Manifest

	man := []byte(`web:
  build: .
  links:
    - postgres
  ports:
    - 5000:3000
  volumes:
    - .:/app
postgres:
  image: convox/postgres
`)

	err := yaml.Unmarshal(man, &manifest)

	if err != nil {
		t.Errorf("ERROR %v", err)
	}

	data, err := buildTemplate("staging", "formation", func() string { return "12345" }, manifest)

	if err != nil {
		t.Errorf("ERROR %v", err)
	}

	var template Template

	if err := json.Unmarshal([]byte(data), &template); err != nil {
		t.Errorf("ERROR %v", err)
	}

	cases := Cases{
		{data, wantedTemplateData()},
		{template.AWSTemplateFormatVersion, "2010-09-09"},
		{template.Parameters["WebPort5000Balancer"]["Default"], "5000"},
		{template.Parameters["WebPort5000Host"]["Default"], "12345"},
		{template.Resources["TaskDefinition"].Type, "Custom::ECSTaskDefinition"},
		{template.Resources["TaskDefinition"].Properties["Environment"], map[string]string{"Ref": "Environment"}},
	}

	_assert(t, cases)
}

func _assert(t *testing.T, cases Cases) {
	for _, c := range cases {
		j1, err := json.Marshal(c.got)

		if err != nil {
			t.Errorf("Marshal %q, error %q", c.got, err)
		}

		j2, err := json.Marshal(c.want)

		if err != nil {
			t.Errorf("Marshal %q, error %q", c.want, err)
		}

		if !bytes.Equal(j1, j2) {
			t.Errorf("Got %q, want %q", c.got, c.want)
		}
	}
}

func wantedTemplateData() string {
	return "\n  {\n    \"AWSTemplateFormatVersion\" : \"2010-09-09\",\n    \"Conditions\": {\n      \n  \n    \"BlankPostgresService\": { \"Fn::Equals\": [ { \"Ref\": \"PostgresService\" }, \"\" ] },\n  \n    \"BlankWebService\": { \"Fn::Equals\": [ { \"Ref\": \"WebService\" }, \"\" ] },\n  \n\n      \"BlankCluster\": { \"Fn::Equals\": [ { \"Ref\": \"Cluster\" }, \"\" ] }\n    },\n    \"Parameters\" : {\n      \n  \n    \"Check\": {\n      \"Type\": \"String\",\n      \"Default\": \"TCP:12345\",\n      \"Description\": \"\"\n    },\n    \n      \n    \n      \n        \n        \"WebPort5000Balancer\": {\n          \"Type\" : \"String\",\n          \"Default\" : \"5000\",\n          \"Description\" : \"\"\n        },\n        \"WebPort5000Host\": {\n          \"Type\" : \"String\",\n          \"Default\" : \"12345\",\n          \"Description\" : \"\"\n        },\n      \n    \n  \n\n      \n  \n    \"PostgresCommand\": {\n      \"Type\" : \"String\",\n      \"Default\" : \"\",\n      \"Description\" : \"\"\n    },\n    \"PostgresImage\": {\n      \"Type\" : \"String\",\n      \"Default\" : \"\",\n      \"Description\" : \"\"\n    },\n    \"PostgresService\": {\n      \"Type\" : \"String\",\n      \"Default\" : \"\",\n      \"Description\" : \"\"\n    },\n  \n    \"WebCommand\": {\n      \"Type\" : \"String\",\n      \"Default\" : \"\",\n      \"Description\" : \"\"\n    },\n    \"WebImage\": {\n      \"Type\" : \"String\",\n      \"Default\" : \"\",\n      \"Description\" : \"\"\n    },\n    \"WebService\": {\n      \"Type\" : \"String\",\n      \"Default\" : \"\",\n      \"Description\" : \"\"\n    },\n  \n\n\n      \"Cluster\": {\n        \"Type\" : \"String\",\n        \"Default\" : \"\",\n        \"Description\" : \"\"\n      },\n      \"Cpu\": {\n        \"Type\": \"Number\",\n        \"Default\": \"200\",\n        \"Description\": \"CPU shares of each process\"\n      },\n      \"DesiredCount\": {\n        \"Type\" : \"Number\",\n        \"Default\" : \"1\",\n        \"Description\" : \"The number of instantiations of the specified ECS task definition to place and keep running on your cluster.\"\n      },\n      \"Environment\": {\n        \"Type\": \"String\",\n        \"Default\": \"\",\n        \"Description\": \"\"\n      },\n      \"Key\": {\n        \"Type\": \"String\",\n        \"Default\": \"\",\n        \"Description\": \"\"\n      },\n      \"Kernel\": {\n        \"Type\" : \"String\",\n        \"Default\" : \"\",\n        \"Description\" : \"\"\n      },\n      \"Memory\": {\n        \"Type\": \"Number\",\n        \"Default\": \"256\",\n        \"Description\": \"MB of RAM to reserve\"\n      },\n      \"Release\": {\n        \"Type\" : \"String\",\n        \"Default\" : \"\",\n        \"Description\" : \"\"\n      },\n      \"Repository\": {\n        \"Type\" : \"String\",\n        \"Default\" : \"\",\n        \"Description\" : \"Source code repository\"\n      },\n      \"Subnets\": {\n        \"Type\" : \"List<AWS::EC2::Subnet::Id>\",\n        \"Default\" : \"\",\n        \"Description\" : \"VPC subnets for this app\"\n      },\n      \"VPC\": {\n        \"Type\" : \"AWS::EC2::VPC::Id\",\n        \"Default\" : \"\",\n        \"Description\" : \"VPC for this app\"\n      }\n    },\n    \"Resources\": {\n      \n  \n    \"BalancerSecurityGroup\": {\n      \"Type\": \"AWS::EC2::SecurityGroup\",\n      \"Properties\": {\n        \"GroupDescription\": { \"Fn::Join\": [ \" \", [ { \"Ref\": \"AWS::StackName\" }, \"-balancer\" ] ] },\n        \"SecurityGroupIngress\": [ { \"CidrIp\": \"0.0.0.0/0\", \"IpProtocol\": \"tcp\", \"FromPort\": { \"Ref\": \"WebPort5000Balancer\" }, \"ToPort\": { \"Ref\": \"WebPort5000Balancer\" } } ],\n        \"VpcId\": { \"Ref\": \"VPC\" }\n      }\n    },\n    \"Balancer\": {\n      \"Type\": \"AWS::ElasticLoadBalancing::LoadBalancer\",\n      \"Properties\": {\n        \"Subnets\": { \"Ref\": \"Subnets\" },\n        \"ConnectionDrainingPolicy\": { \"Enabled\": true, \"Timeout\": 60 },\n        \"ConnectionSettings\": { \"IdleTimeout\": 60 },\n        \"CrossZone\": true,\n        \"HealthCheck\": {\n          \"HealthyThreshold\": \"2\",\n          \"Interval\": 5,\n          \"Target\": { \"Ref\": \"Check\" },\n          \"Timeout\": 3,\n          \"UnhealthyThreshold\": \"2\"\n        },\n        \"Listeners\": [ { \"Protocol\": \"TCP\", \"LoadBalancerPort\": { \"Ref\": \"WebPort5000Balancer\" }, \"InstanceProtocol\": \"TCP\", \"InstancePort\": { \"Ref\": \"WebPort5000Host\" } } ],\n        \"LBCookieStickinessPolicy\": [{ \"PolicyName\": \"affinity\" }],\n        \"LoadBalancerName\": { \"Ref\": \"AWS::StackName\" },\n        \"SecurityGroups\": [ { \"Ref\": \"BalancerSecurityGroup\" } ]\n      }\n    },\n  \n\n      \n  \"Kinesis\": {\n    \"Type\": \"AWS::Kinesis::Stream\",\n    \"Properties\": {\n      \"ShardCount\": 1\n    }\n  },\n  \n    \"LogsUser\": {\n      \"Type\": \"AWS::IAM::User\",\n      \"Properties\": {\n        \"Path\": \"/convox/\",\n        \"Policies\": [\n          {\n            \"PolicyName\": \"LogsRole\",\n            \"PolicyDocument\": {\n              \"Version\": \"2012-10-17\",\n              \"Statement\": [\n                {\n                  \"Effect\": \"Allow\",\n                  \"Action\": [ \"kinesis:PutRecords\" ],\n                  \"Resource\": [ { \"Fn::Join\": [ \"\", [ \"arn:aws:kinesis:*:*:stream/\", { \"Ref\": \"AWS::StackName\" }, \"-*\" ] ] } ]\n                }\n              ]\n            }\n          }\n        ]\n      }\n    },\n    \"LogsAccess\": {\n      \"Type\": \"AWS::IAM::AccessKey\",\n      \"Properties\": {\n        \"Serial\": \"1\",\n        \"Status\": \"Active\",\n        \"UserName\": { \"Ref\": \"LogsUser\" }\n      }\n    },\n    \"TaskDefinition\": {\n      \"Type\": \"Custom::ECSTaskDefinition\",\n      \"Version\": \"1.0\",\n      \"Properties\": {\n        \"ServiceToken\": { \"Ref\": \"Kernel\" },\n        \"Name\": { \"Ref\": \"AWS::StackName\" },\n        \"Release\": { \"Ref\": \"Release\" },\n        \"Environment\": { \"Ref\": \"Environment\" },\n        \"Tasks\": [\n          { \"Fn::If\": [ \"BlankWebService\",\n\t\t\t\t{\n\t\t\t\t\t\"Name\": \"web\",\n\t\t\t\t\t\"Image\": { \"Ref\": \"WebImage\" },\n\t\t\t\t\t\"Command\": { \"Ref\": \"WebCommand\" },\n\t\t\t\t\t\"Key\": { \"Ref\": \"Key\" },\n\t\t\t\t\t\"CPU\": { \"Ref\": \"Cpu\" },\n\t\t\t\t\t\"Memory\": { \"Ref\": \"Memory\" },\n\t\t\t\t\t\"Environment\": {\n\t\t\t\t\t\t\"KINESIS\": { \"Ref\": \"Kinesis\" },\n\t\t\t\t\t\t\"PROCESS\": \"web\"\n\t\t\t\t\t},\n\t\t\t\t\t\"Links\": [ { \"Fn::If\": [ \"BlankPostgresService\",\n\t\t\t\t\t\t\"postgres:postgres\",\n\t\t\t\t\t\t{ \"Ref\" : \"AWS::NoValue\" } ] } ],\n\t\t\t\t\t\"Volumes\": [  ],\n\t\t\t\t\t\"Services\": [ { \"Fn::If\": [ \"BlankPostgresService\",\n\t\t\t\t\t\t{ \"Ref\" : \"AWS::NoValue\" },\n\t\t\t\t\t\t{ \"Fn::Join\": [ \":\", [ { \"Ref\" : \"PostgresService\" }, \"postgres\" ] ] } ] } ],\n\t\t\t\t\t\"PortMappings\": [ { \"Fn::Join\": [ \":\", [ { \"Ref\": \"WebPort5000Host\" }, \"3000\" ] ] } ]\n\t\t\t\t}, { \"Ref\" : \"AWS::NoValue\" } ] },{ \"Fn::If\": [ \"BlankPostgresService\",\n\t\t\t\t{\n\t\t\t\t\t\"Name\": \"postgres\",\n\t\t\t\t\t\"Image\": { \"Ref\": \"PostgresImage\" },\n\t\t\t\t\t\"Command\": { \"Ref\": \"PostgresCommand\" },\n\t\t\t\t\t\"Key\": { \"Ref\": \"Key\" },\n\t\t\t\t\t\"CPU\": { \"Ref\": \"Cpu\" },\n\t\t\t\t\t\"Memory\": { \"Ref\": \"Memory\" },\n\t\t\t\t\t\"Environment\": {\n\t\t\t\t\t\t\"KINESIS\": { \"Ref\": \"Kinesis\" },\n\t\t\t\t\t\t\"PROCESS\": \"postgres\"\n\t\t\t\t\t},\n\t\t\t\t\t\"Links\": [  ],\n\t\t\t\t\t\"Volumes\": [  ],\n\t\t\t\t\t\"Services\": [  ],\n\t\t\t\t\t\"PortMappings\": [  ]\n\t\t\t\t}, { \"Ref\" : \"AWS::NoValue\" } ] }\n        ]\n      }\n    },\n    \"Service\": {\n      \"Type\": \"Custom::ECSService\",\n      \"Version\": \"1.0\",\n      \"Properties\": {\n        \"ServiceToken\": { \"Ref\": \"Kernel\" },\n        \"Cluster\": { \"Ref\": \"Cluster\" },\n        \"DesiredCount\": { \"Ref\": \"DesiredCount\" },\n        \"Name\": { \"Ref\": \"AWS::StackName\" },\n        \"TaskDefinition\": { \"Ref\": \"TaskDefinition\" },\n        \"Role\": { \"Ref\": \"ServiceRole\" },\n        \"LoadBalancers\": [ { \"Fn::Join\": [ \":\", [ { \"Ref\": \"Balancer\" }, \"web\", \"3000\" ] ] } ]\n      }\n    },\n  \n\n\n      \n  \"ServiceRole\": {\n    \"Type\": \"AWS::IAM::Role\",\n    \"Properties\": {\n      \"AssumeRolePolicyDocument\": {\n        \"Statement\": [\n          {\n            \"Action\": [\n              \"sts:AssumeRole\"\n            ],\n            \"Effect\": \"Allow\",\n            \"Principal\": {\n              \"Service\": [\n                \"ecs.amazonaws.com\"\n              ]\n            }\n          }\n        ],\n        \"Version\": \"2012-10-17\"\n      },\n      \"Path\": \"/\",\n      \"Policies\": [\n        {\n          \"PolicyName\": \"ServiceRole\",\n          \"PolicyDocument\": {\n            \"Statement\": [\n              {\n                \"Effect\": \"Allow\",\n                \"Action\": [\n                  \"elasticloadbalancing:Describe*\",\n                  \"elasticloadbalancing:DeregisterInstancesFromLoadBalancer\",\n                  \"elasticloadbalancing:RegisterInstancesWithLoadBalancer\",\n                  \"ec2:Describe*\",\n                  \"ec2:AuthorizeSecurityGroupIngress\"\n                ],\n                \"Resource\": [\n                  \"*\"\n                ]\n              }\n            ]\n          }\n        }\n      ]\n    }\n  },\n\n      \n  \"Settings\": {\n    \"Type\": \"AWS::S3::Bucket\",\n    \"Properties\": {\n      \"AccessControl\": \"Private\",\n      \"Tags\": [\n        { \"Key\": \"system\", \"Value\": \"convox\" },\n        { \"Key\": \"app\", \"Value\": { \"Ref\": \"AWS::StackName\" } }\n      ]\n    }\n  }\n\n    },\n    \"Outputs\": {\n      \n  \n    \"BalancerHost\": {\n      \"Value\": { \"Fn::GetAtt\": [ \"Balancer\", \"DNSName\" ] }\n    },\n  \n  \n    \n  \n    \n      \n        \n        \"WebPort5000Balancer\": {\n          \"Value\": { \"Ref\": \"WebPort5000Balancer\" }\n        },\n      \n    \n  \n\n      \n  \"Kinesis\": {\n    \"Value\": { \"Ref\": \"Kinesis\" }\n  },\n\n\n      \"Settings\": {\n        \"Value\": { \"Ref\": \"Settings\" }\n      }\n    }\n  }\n"
}
