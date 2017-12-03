package sparta

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	spartaAWS "github.com/mweagle/Sparta/aws"
)

// Dynamically assigned discover function that is set by Main
var discoverImpl func() (*DiscoveryInfo, error)

var discoveryCache map[string]*DiscoveryInfo

////////////////////////////////////////////////////////////////////////////////
// START - DiscoveryResource
//

// DiscoveryResource stores information about a CloudFormation resource
// that the calling Go function `DependsOn`.
type DiscoveryResource struct {
	ResourceID string
	Properties map[string]string
	Tags       map[string]string
}

func newDiscoveryResource(resourceID string, props map[string]interface{}) (DiscoveryResource, error) {
	resource := DiscoveryResource{}
	resource.ResourceID = resourceID
	resource.Properties = make(map[string]string, 0)
	resource.Tags = make(map[string]string, 0)

	for eachProp, eachValue := range props {
		if eachProp != "Tags" {
			assertedValue, assertOK := eachValue.(string)
			if !assertOK {
				return resource, fmt.Errorf("Invalid type assertion for newDiscoveryResource factory: %s=>%#v", eachProp, eachValue)
			}
			resource.Properties[eachProp] = assertedValue
		} else {
			tagArray, ok := eachValue.([]interface{})
			if !ok {
				return resource, fmt.Errorf("Failed to type asset Tags")
			}
			for _, eachEntry := range tagArray {
				eachTagMap, ok := eachEntry.(map[string]interface{})
				if !ok {
					return resource, fmt.Errorf("Failed to type asset tag pair: %#v", eachTagMap)
				}
				tagName, keyOK := eachTagMap["Key"].(string)
				tagValue, valueOK := eachTagMap["Value"].(string)
				if !keyOK || !valueOK {
					return resource, errors.New("Failed to unmarshal tag")
				}
				resource.Tags[tagName] = tagValue
			}
		}
	}
	return resource, nil
}

//
// END - DiscoveryResource
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// START - DiscoveryInfo
//

// DiscoveryInfo encapsulates information returned by `sparta.Discovery()`
// to enable a runtime function to discover information about its
// AWS environment or resources that the function created explicit
// `DependsOn` relationships
type DiscoveryInfo struct {
	// Current logical resource ID
	ResourceID string
	// Current AWS region
	Region string
	// Current Stack ID
	StackID string
	// StackName (eg, Sparta service name)
	StackName string
	// Map of resources this Go function has explicit `DependsOn` relationship
	Resources map[string]DiscoveryResource
}

// UnmarshalJSON is responsible for transforming the raw discovery data into
func (discoveryInfo *DiscoveryInfo) UnmarshalJSON(data []byte) error {
	var discoveryData map[string]interface{}
	if err := json.Unmarshal(data, &discoveryData); err != nil {
		return err
	}
	discoveryInfo.Resources = make(map[string]DiscoveryResource, 0)
	for eachKey, eachValue := range discoveryData {
		typeAssertOk := true

		switch eachKey {
		case TagLogicalResourceID:
			discoveryInfo.ResourceID, typeAssertOk = eachValue.(string)
		case TagStackRegion:
			discoveryInfo.Region, typeAssertOk = eachValue.(string)
		case TagStackID:
			discoveryInfo.StackID, typeAssertOk = eachValue.(string)
		case TagStackName:
			discoveryInfo.StackName, typeAssertOk = eachValue.(string)
		case "golangFunc":
			// NOP
		default:
			assertValue, assertOK := eachValue.(map[string]interface{})
			if !assertOK {
				return fmt.Errorf("Failed to type assert entry: %s=>%#v", eachKey, eachValue)
			}
			res, err := newDiscoveryResource(eachKey, assertValue)
			if nil != err {
				return err
			}
			discoveryInfo.Resources[eachKey] = res
		}
		if !typeAssertOk {
			err := fmt.Errorf("Failed to create resource")
			return err
		}
	}
	return nil
}

//
// START - DiscoveryInfo
////////////////////////////////////////////////////////////////////////////////

// Discover returns metadata information for resources upon which
// the current golang lambda function depends.
func Discover() (*DiscoveryInfo, error) {
	if nil == discoverImpl {
		return nil, fmt.Errorf("Discovery service has not been initialized")
	}
	return discoverImpl()
}

// TODO cache the data somewhere other than querying CF
func initializeDiscovery(serviceName string, lambdaAWSInfos []*LambdaAWSInfo, logger *logrus.Logger) {
	// Setup the discoveryImpl reference
	discoveryCache = make(map[string]*DiscoveryInfo, 0)
	discoverImpl = func() (*DiscoveryInfo, error) {
		pc := make([]uintptr, 2)
		entriesWritten := runtime.Callers(2, pc)
		if entriesWritten != 2 {
			return nil, fmt.Errorf("Unsupported call site for sparta.Discover()")
		}

		// The actual caller is sparta.Discover()
		f := runtime.FuncForPC(pc[1])
		golangFuncName := f.Name()

		// Find the LambdaAWSInfo that has this golang function
		// as its target
		lambdaCFResource := ""
		for _, eachLambda := range lambdaAWSInfos {
			if eachLambda.lambdaFnName == golangFuncName {
				lambdaCFResource = eachLambda.logicalName()
			}
		}
		logger.WithFields(logrus.Fields{
			"CallerName":     golangFuncName,
			"CFResourceName": lambdaCFResource,
			"ServiceName":    serviceName,
		}).Debug("Discovery Info")
		if "" == lambdaCFResource {
			return nil, fmt.Errorf("Unsupported call site for sparta.Discover(): %s", golangFuncName)
		}

		emptyConfiguration := &DiscoveryInfo{}
		if "" != lambdaCFResource {
			cachedConfig, exists := discoveryCache[lambdaCFResource]
			if exists {
				return cachedConfig, nil
			}

			// Look it up
			awsCloudFormation := cloudformation.New(spartaAWS.NewSession(logger))
			params := &cloudformation.DescribeStackResourceInput{
				LogicalResourceId: aws.String(lambdaCFResource),
				StackName:         aws.String(serviceName),
			}
			result, err := awsCloudFormation.DescribeStackResource(params)
			if nil != err {
				// TODO - retry/cache expiry
				discoveryCache[lambdaCFResource] = emptyConfiguration
				return nil, err
			}
			metadata := result.StackResourceDetail.Metadata
			if nil == metadata {
				metadata = aws.String("{}")
			}

			// Transform this into a map
			logger.WithFields(logrus.Fields{
				"Metadata": metadata,
			}).Debug("DiscoveryInfo Metadata")
			var discoveryInfo DiscoveryInfo
			err = json.Unmarshal([]byte(*metadata), &discoveryInfo)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"Metadata": *metadata,
					"Error":    err,
				}).Error("Failed to unmarshal discovery info")
			}
			discoveryCache[lambdaCFResource] = &discoveryInfo
			return &discoveryInfo, err
		}
		return emptyConfiguration, nil
	}
}
