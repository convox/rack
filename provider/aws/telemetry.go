package aws

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

var (
	skipParams = strings.Join([]string{
		"DefaultAmi",
		"DefaultAmiArm",
		"Telemetry",
	}, ",")

	redactedParams = strings.Join([]string{
		"Ami",
		"ApiCount",
		"ApiCpu",
		"ApiMonitorMemory",
		"ApiRouter",
		"ApiWebMemory",
		"Autoscale",
		"AvailabilityZones",
		"BuildCpu",
		"BuildImage",
		"BuildInstance",
		"BuildInstancePolicy",
		"BuildMemory",
		"BuildVolumeSize",
		"ClientId",
		"Development",
		"EcsPollInterval",
		"EncryptEbs",
		"Encryption",
		"ExistingVpc",
		"HighAvailability",
		"HttpProxy",
		"ImagePullBehavior",
		"IMDSHttpTokens",
		"Internal",
		"InternalOnly",
		"InternalSuffix",
		"InstanceBootCommand",
		"InstanceRunCommand",
		"InstanceType",
		"InstanceUpdateBatchSize",
		"InstancePolicy",
		"InstanceSecurityGroup",
		"BuildInstanceSecurityGroup",
		"InternetGateway",
		"Key",
		"LogBucket",
		"LogDriver",
		"LogRetention",
		"Password",
		"Private",
		"PrivateApi",
		"PrivateApiSecurityGroup",
		"PrivateBuild",
		"RouterInternalSecurityGroup",
		"RouterSecurityGroup",
		"ScheduleRackScaleDown",
		"ScheduleRackScaleUp",
		"SpotInstanceBid",
		"SslPolicy",
		"Subnet0CIDR",
		"Subnet1CIDR",
		"Subnet2CIDR",
		"SubnetPrivate0CIDR",
		"SubnetPrivate1CIDR",
		"SubnetPrivate2CIDR",
		"SwapSize",
		"SyslogDestination",
		"SyslogFormat",
		"Version",
		"VPCCIDR",
		"Tenancy",
		"WhiteList",
	}, ",")
)

func (p *Provider) RackParamsToSync(params map[string]string) map[string]string {
	toSync := make(map[string]string)
	for k, v := range params {
		if !strings.Contains(skipParams, k) {
			if strings.Contains(redactedParams, k) {
				toSync[k] = hashParamValue(v)
			} else {
				toSync[k] = v
			}
		}
	}

	return toSync
}

func hashParamValue(value string) string {
	hasher := sha256.New()
	hasher.Write([]byte(value))
	return hex.EncodeToString(hasher.Sum(nil))
}
