package sparta

import (
	"encoding/json"
	"testing"
)

const discoveryDataTags = `
{
  "SESMessageStoreBucketa622fdfda5789d596c08c79124f12b978b3da772": {
    "DomainName": "spartaapplication-sesmessagestorebucketa622fdfda5-1b8t1fol64if3.s3.amazonaws.com",
    "Ref": "spartaapplication-sesmessagestorebucketa622fdfda5-1b8t1fol64if3",
    "Tags": [
      {
        "Key": "sparta:logicalBucketName",
        "Value": "Special"
      }
    ],
    "Type": "AWS::S3::Bucket",
    "WebsiteURL": "http://spartaapplication-sesmessagestorebucketa622fdfda5-1b8t1fol64if3.s3-website-us-west-2.amazonaws.com"
  },
  "aws:cloudformation:stack-id": "arn:aws:cloudformation:us-west-2:123412341234:stack/SpartaApplication/c25fab50-c904-11e5-acca-503f20f2ade6",
  "aws:cloudformation:stack-name": "SpartaApplication",
  "golangFunc": "main.echoSESEvent",
  "sparta:cloudformation:region": "us-west-2"
}
`

var discoveryDataNoTags = `
{
  "S3DynamicBucket5de4436284814c262da3b904c1f3fc73b23cea00": {
    "DomainName": "spartaapplication-s3dynamicbucket5de4436284814c26-ll4cejoliisg.s3.amazonaws.com",
    "Ref": "spartaapplication-s3dynamicbucket5de4436284814c26-ll4cejoliisg",
    "Type": "AWS::S3::Bucket",
    "WebsiteURL": "http://spartaapplication-s3dynamicbucket5de4436284814c26-ll4cejoliisg.s3-website-us-west-2.amazonaws.com"
  },
  "aws:cloudformation:stack-id": "arn:aws:cloudformation:us-west-2:123412341234:stack/SpartaApplication/c25fab50-c904-11e5-acca-503f20f2ade6",
  "aws:cloudformation:stack-name": "SpartaApplication",
  "golangFunc": "main.echoS3DynamicBucketEvent",
  "sparta:cloudformation:region": "us-west-2"
}
`

func TestDiscoveryInitialized(t *testing.T) {
	// Ensure that sparta.Discover() can only be called from a lambda function
	logger, _ := NewLogger("warning")
	initializeDiscovery("DiscoveryTest", testLambdaData(), logger)

	configuration, err := Discover()
	t.Logf("Configuration: %#v", configuration)
	t.Logf("Error: %#v", err)
	if err == nil {
		t.Errorf("sparta.Discover() failed to reject invalid call site")
	}
	t.Logf("Properly rejected invalid callsite: %s", err.Error())
}

func TestDiscoveryNotInitialized(t *testing.T) {
	// Ensure that sparta.Discover() can only be called from a lambda function
	configuration, err := Discover()
	t.Logf("Configuration: %#v", configuration)
	t.Logf("Error: %#v", err)
	if err == nil {
		t.Errorf("sparta.Discover() failed to error when not initialized")
	}
	t.Logf("Properly rejected invalid callsite: %s", err.Error())
}

func TestDiscoveryUnmarshalTags(t *testing.T) {
	// Ensure that sparta.Discover() can only be called from a lambda function
	var info DiscoveryInfo
	err := json.Unmarshal([]byte(discoveryDataTags), &info)
	if nil != err {
		t.Errorf("Failed to unmarshal discovery data with tags")
	}
	if 1 != len(info.Resources) {
		t.Errorf("Failed to unmarshal single resource")
	}
	if 1 != len(info.Resources["SESMessageStoreBucketa622fdfda5789d596c08c79124f12b978b3da772"].Tags) {
		t.Errorf("Invalid Tags unmarshaled resource with single tag")
	}
	if info.Resources["SESMessageStoreBucketa622fdfda5789d596c08c79124f12b978b3da772"].Tags["sparta:logicalBucketName"] != "Special" {
		t.Errorf("Failed to unmarshal `sparta:logicalBucketName` tag")
	}
	t.Logf("Discovery Info: %#v", info)
}

func TestDiscoveryUnmarshalNoTags(t *testing.T) {
	// Ensure that sparta.Discover() can only be called from a lambda function
	var info DiscoveryInfo
	err := json.Unmarshal([]byte(discoveryDataNoTags), &info)
	if nil != err {
		t.Errorf("Failed to unmarshal discovery data without tags")
	}
	if 1 != len(info.Resources) {
		t.Errorf("Failed to unmarshal single resource")
	}
	if 0 != len(info.Resources["S3DynamicBucket5de4436284814c262da3b904c1f3fc73b23cea00"].Tags) {
		t.Errorf("Invalid Tags unmarshaled for tagless resource")
	}
	t.Logf("Discovery Info: %#v", info)
}

func TestDiscoveryEmptyMetadata(t *testing.T) {
	// Ensure that sparta.Discover() can only be called from a lambda function
	var info DiscoveryInfo
	err := json.Unmarshal([]byte("{}"), &info)
	if nil != err {
		t.Errorf("Failed to unmarshal empty discovery data")
	}
	t.Logf("Discovery Info: %#v", info)
}
