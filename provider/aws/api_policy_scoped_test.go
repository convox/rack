package aws

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type scopedPolicyRack struct {
	Conditions map[string]json.RawMessage
	Parameters map[string]json.RawMessage
	Resources  map[string]struct {
		Type       string
		Condition  string
		Properties json.RawMessage
	}
}

var scopedPolicyExpectedActions = []string{
	"acm:*",
	"application-autoscaling:*",
	"autoscaling:*",
	"cloudformation:*",
	"cloudwatch:*",
	"dynamodb:*",
	"ec2:*",
	"ecr:*",
	"ecs:*",
	"elasticache:*",
	"elasticfilesystem:*",
	"elasticloadbalancing:*",
	"events:*",
	"iam:CreateServiceLinkedRole",
	"iam:ListRoles",
	"kms:*",
	"lambda:*",
	"logs:*",
	"managed-fleets:*",
	"rds:*",
	"route53:*",
	"s3:*",
	"sns:*",
	"sqs:*",
	"ssm:*",
	"sts:GetServiceBearerToken",
}

var scopedPolicyCloudFormationServices = map[string]string{
	"ApplicationAutoScaling": "application-autoscaling",
	"AutoScaling":            "autoscaling",
	"CertificateManager":     "acm",
	"CloudFormation":         "cloudformation",
	"DynamoDB":               "dynamodb",
	"EC2":                    "ec2",
	"ECR":                    "ecr",
	"ECS":                    "ecs",
	"EFS":                    "elasticfilesystem",
	"ElastiCache":            "elasticache",
	"ElasticLoadBalancing":   "elasticloadbalancing",
	"ElasticLoadBalancingV2": "elasticloadbalancing",
	"Events":                 "events",
	"IAM":                    "iam",
	"KMS":                    "kms",
	"Lambda":                 "lambda",
	"Logs":                   "logs",
	"RDS":                    "rds",
	"Route53":                "route53",
	"S3":                     "s3",
	"SNS":                    "sns",
	"SQS":                    "sqs",
	"SSM":                    "ssm",
}

var scopedPolicySDKClients = map[string]string{
	"cloudwatchlogs": "logs",
	"eventbridge":    "events",
}

var scopedPolicyUnusedProperties = map[string]string{
	"AuthenticateCognitoConfig":  "cognito-idp",
	"CertificateAuthorityArn":    "acm-pca",
	"DataProtectionPolicy":       "firehose",
	"FileSystemConfigs":          "s3files",
	"KinesisStreamSpecification": "kinesis",
	"LogDeliveryConfigurations":  "firehose",
	"ManageMasterUserPassword":   "secretsmanager",
	"MetadataConfiguration":      "s3tables",
	"MetadataTableConfiguration": "s3tables",
	"TrafficSources":             "vpc-lattice",
}

func TestApiRoleScopedWiring(t *testing.T) {
	rack := scopedPolicyLoadRack(t)

	var param struct {
		Type          string
		Default       string
		AllowedValues []string
	}
	scopedPolicyDecode(t, "ApiRoleScoped parameter", rack.Parameters["ApiRoleScoped"], &param)
	if param.Type != "String" || param.Default != "No" || !reflect.DeepEqual(param.AllowedValues, []string{"Yes", "No"}) {
		t.Fatalf("ApiRoleScoped parameter: %+v", param)
	}

	var cond, wantCond interface{}
	scopedPolicyDecode(t, "ApiRoleScopedEnabled condition", rack.Conditions["ApiRoleScopedEnabled"], &cond)
	scopedPolicyDecode(t, "expected condition", []byte(`{"Fn::Equals": [{"Ref": "ApiRoleScoped"}, "Yes"]}`), &wantCond)
	if !reflect.DeepEqual(cond, wantCond) {
		t.Fatalf("ApiRoleScopedEnabled condition: %v", cond)
	}

	policy := rack.Resources["ApiPolicyScoped"]
	if policy.Type != "AWS::IAM::ManagedPolicy" || policy.Condition != "ApiRoleScopedEnabled" {
		t.Fatalf("ApiPolicyScoped type %q condition %q", policy.Type, policy.Condition)
	}
	var props map[string]json.RawMessage
	scopedPolicyDecode(t, "ApiPolicyScoped properties", policy.Properties, &props)
	keys := []string{}
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if !reflect.DeepEqual(keys, []string{"Path", "PolicyDocument"}) || string(props["Path"]) != `"/convox/"` {
		t.Fatalf("ApiPolicyScoped properties %v, Path %s", keys, props["Path"])
	}

	var role struct{ ManagedPolicyArns interface{} }
	var wantArns interface{}
	scopedPolicyDecode(t, "ApiRole properties", rack.Resources["ApiRole"].Properties, &role)
	scopedPolicyDecode(t, "expected ManagedPolicyArns", []byte(`[
		{"Fn::If": ["ApiRoleScopedEnabled", {"Ref": "ApiPolicyScoped"}, {"Fn::Sub": "arn:${AWS::Partition}:iam::aws:policy/PowerUserAccess"}]},
		{"Ref": "ApiPolicyV2"},
		{"Ref": "CMKPolicy"}
	]`), &wantArns)
	if !reflect.DeepEqual(role.ManagedPolicyArns, wantArns) {
		t.Fatalf("ApiRole ManagedPolicyArns: %v", role.ManagedPolicyArns)
	}
}

func TestApiPolicyScopedActions(t *testing.T) {
	actions := append([]string{}, scopedPolicyActions(t, scopedPolicyLoadRack(t))...)
	sort.Strings(actions)
	if !reflect.DeepEqual(actions, scopedPolicyExpectedActions) {
		t.Fatalf("ApiPolicyScoped actions:\n got %v\nwant %v", actions, scopedPolicyExpectedActions)
	}
}

// With ApiRoleScoped=Yes, every template type, SDK client, and secret or STS call must map to a grant ApiRole still holds.
func TestApiPolicyScopedCoverage(t *testing.T) {
	rack := scopedPolicyLoadRack(t)

	// IAM actions come from ApiPolicyV2.
	granted := map[string]bool{"iam": true}
	for _, a := range scopedPolicyActions(t, rack) {
		if ns, ok := strings.CutSuffix(a, ":*"); ok {
			granted[ns] = true
		}
	}

	types := regexp.MustCompile(`AWS::([A-Za-z0-9]+)::`)
	seenTypes := map[string]bool{}
	for path, text := range scopedPolicyFiles(t, []string{"formation", "templates"}, func(string) bool { return true }) {
		for _, m := range types.FindAllStringSubmatch(text, -1) {
			seenTypes[m[1]] = true
			if !granted[scopedPolicyCloudFormationServices[m[1]]] {
				t.Errorf("%s: AWS::%s:: types have no namespace in ApiPolicyScoped", path, m[1])
			}
		}
		for prop, ns := range scopedPolicyUnusedProperties {
			if strings.Contains(text, `"`+prop+`"`) {
				t.Errorf("%s: %s needs %s, which ApiPolicyScoped does not grant", path, prop, ns)
			}
		}
	}
	if !seenTypes["EC2"] || !seenTypes["CloudFormation"] {
		t.Fatalf("template scan missed EC2 or CloudFormation types: %v", seenTypes)
	}

	goSource := func(path string) bool { return strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") }

	imports := regexp.MustCompile(`"github\.com/aws/aws-sdk-go(?:-v2)?/service/([a-z0-9]+)"`)
	seenClients := map[string]bool{}
	for path, text := range scopedPolicyFiles(t, []string{"..", "../../pkg", "../../cmd"}, goSource) {
		for _, m := range imports.FindAllStringSubmatch(text, -1) {
			client := m[1]
			seenClients[client] = true
			if client == "secretsmanager" || client == "sts" {
				if filepath.Dir(path) != filepath.Join("..", "aws") {
					t.Errorf("%s: %s calls are only checked in provider/aws", path, client)
				}
				continue
			}
			ns := client
			if v, ok := scopedPolicySDKClients[client]; ok {
				ns = v
			}
			if !granted[ns] {
				t.Errorf("%s: %s client has no namespace in ApiPolicyScoped", path, client)
			}
		}
	}
	if !seenClients["cloudformation"] {
		t.Fatalf("import scan missed the cloudformation client: %v", seenClients)
	}

	calls := regexp.MustCompile(`p\.(secretsmanager|sts)\(\)\.([A-Za-z]+)\(`)
	secrets := scopedPolicyInlineActions(t, rack, "SecretsManagerAccess")
	seenCalls := map[string]bool{}
	for path, text := range scopedPolicyFiles(t, []string{"."}, goSource) {
		for _, m := range calls.FindAllStringSubmatch(text, -1) {
			seenCalls[m[1]] = true
			if (m[1] == "sts" && m[2] != "GetCallerIdentity") || (m[1] == "secretsmanager" && !secrets["secretsmanager:"+m[2]]) {
				t.Errorf("%s: %s:%s is not granted to ApiRole when ApiRoleScoped=Yes", path, m[1], m[2])
			}
		}
	}
	if !seenCalls["secretsmanager"] {
		t.Fatalf("call scan missed secretsmanager calls")
	}
}

func scopedPolicyLoadRack(t *testing.T) scopedPolicyRack {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("formation", "rack.json"))
	if err != nil {
		t.Fatalf("read rack.json: %v", err)
	}
	var rack scopedPolicyRack
	scopedPolicyDecode(t, "rack.json", data, &rack)
	return rack
}

func scopedPolicyDecode(t *testing.T, label string, data []byte, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("decode %s: %v", label, err)
	}
}

func scopedPolicyActions(t *testing.T, rack scopedPolicyRack) []string {
	t.Helper()
	var props struct {
		PolicyDocument struct {
			Version   string
			Statement []struct {
				Effect   string
				Action   []string
				Resource string
			}
		}
	}
	scopedPolicyDecode(t, "ApiPolicyScoped properties", rack.Resources["ApiPolicyScoped"].Properties, &props)
	doc := props.PolicyDocument
	if doc.Version != "2012-10-17" || len(doc.Statement) != 1 || doc.Statement[0].Effect != "Allow" || doc.Statement[0].Resource != "*" {
		t.Fatalf("ApiPolicyScoped document: %+v", doc)
	}
	return doc.Statement[0].Action
}

func scopedPolicyInlineActions(t *testing.T, rack scopedPolicyRack, name string) map[string]bool {
	t.Helper()
	var role struct {
		Policies []struct {
			PolicyName     string
			PolicyDocument struct {
				Statement []struct{ Action []string }
			}
		}
	}
	scopedPolicyDecode(t, "ApiRole properties", rack.Resources["ApiRole"].Properties, &role)
	actions := map[string]bool{}
	for _, p := range role.Policies {
		if p.PolicyName != name {
			continue
		}
		for _, s := range p.PolicyDocument.Statement {
			for _, a := range s.Action {
				actions[a] = true
			}
		}
	}
	if len(actions) == 0 {
		t.Fatalf("ApiRole has no %s policy", name)
	}
	return actions
}

func scopedPolicyFiles(t *testing.T, roots []string, keep func(string) bool) map[string]string {
	t.Helper()
	files := map[string]string{}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !keep(path) {
				return err
			}
			data, err := os.ReadFile(path)
			files[path] = string(data)
			return err
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	return files
}
