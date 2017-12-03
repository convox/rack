## v0.7.1
- :warning: **BREAKING**
- :checkered_flag: **CHANGES**
  - Upgrade to latest [go-cloudformation](https://github.com/crewjam/go-cloudformation) that required internal [refactoring](https://github.com/mweagle/Sparta/pull/9).
- :bug: **FIXED**
  - N/A
## v0.7.0
- :warning: **BREAKING**
  - `TemplateDecorator` signature changed to include `serviceName`, `S3Bucket`, and `S3Key` to allow for decorating CloudFormation with [UserData](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/user-data.html) to support [alternative topology](http://gosparta.io/docs/alternative_topologies/) deployments.
  - `CommonIAMStatements` changed from `map[string][]iamPolicyStatement` to struct with named fields.
  - `PushSourceConfigurationActions` changed from `map[string][]string` to struct with named fields.
  - Eliminated [goptions](https://github.com/voxelbrain/goptions)
- :checkered_flag: **CHANGES**
  - Moved CLI parsing to [Cobra](https://github.com/spf13/cobra)
    - Applications can extend the set of flags for existing Sparta commands (eg, `provision` can include `--subnetIDs`) as well as add their own top level commands to the `CommandLineOptions` exported values.  See [SpartaCICD](https://github.com/mweagle/SpartaCICD) for an example.
  - Added _Sparta/aws/cloudformation_ `ConvertToTemplateExpression` to convert string value into [Fn::Join](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference-join.html) compatible representation. Parses inline AWS references and supports user-defined [template](https://golang.org/pkg/text/template/) properties.
  - Added `sparta/aws/iam` _PolicyStatement_ type
  - Upgraded `describe` output to use [Mermaid 6.0.0](https://github.com/knsv/mermaid/releases/tag/6.0.0)
  - All [goreportcard](https://goreportcard.com/report/github.com/mweagle/Sparta) issues fixed.
- :bug: **FIXED**
  - Fixed latent VPC provisioning bug where VPC/Subnet IDs couldn't be provided to template serialization.
  
## v0.6.0
- :warning: **BREAKING**
  - `TemplateDecorator` signature changed to include `map[string]string` to allow for decorating CloudFormation resource metadata
- :checkered_flag: **CHANGES**
  - All NodeJS CustomResources moved to _go_
  - Add support for user-defined CloudFormation CustomResources via `LambdaAWSInfo.RequireCustomResource`
  - `DiscoveryInfo` struct now includes `TagLogicalResourceID` field with CloudFormation Resource ID of calling lambda function
- :bug: **FIXED**
  - N/A

## v0.5.5
This release includes a major internal refactoring to move the current set of NodeJS [Lambda-backed CloudFormation CustomResources](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/template-custom-resources-lambda.html) to Sparta Go functions. The two migrated CustomActions are:

* The S3 event source configuration
* Provisioning an S3-static site

Both are implemented using [cloudformationresources](https://github.com/mweagle/cloudformationresources). There are no changes to the calling code and no regressions are expected.

- :warning: **BREAKING**
  - APIGateway provisioning now only creates a single discovery file: _MANIFEST.json_ at the site root.
- :checkered_flag: **CHANGES**
  - VPC support! Added [LambdaFunctionVPCConfig](https://godoc.org/github.com/crewjam/go-cloudformation#LambdaFunctionVPCConfig) to [LambdaFunctionsOptions](https://godoc.org/github.com/mweagle/Sparta#LambdaFunctionOptions) struct.
  - Updated NodeJS runtime to [nodejs4.3](http://docs.aws.amazon.com/lambda/latest/dg/programming-model.html)
  - CloudFormation updates are now done via [Change Sets](https://aws.amazon.com/blogs/aws/new-change-sets-for-aws-cloudformation/), rather than [UpdateStack](http://docs.aws.amazon.com/sdk-for-go/api/service/cloudformation/CloudFormation.html#UpdateStack-instance_method).
  - APIGateway and CloudWatchEvents are now configured using [CloudFormation](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/ReleaseHistory.html). They were previously implemented using NodeJS CustomResources.
- :bug: **FIXED**
  - Fixed latent issue where `IAM::Role` resources didn't use stable CloudFormation resource names
  - Fixed latent issue where names & descriptions of Lambda functions weren't consistent
  - https://github.com/mweagle/SpartaApplication/issues/1

## v0.5.4
- :warning: **BREAKING**
  - N/A
- :checkered_flag: **CHANGES**
  - Run `go generate` as part of the _provision_ step
- :bug: **FIXED**
  - N/A

## v0.5.3
- :warning: **BREAKING**
  - N/A
- :checkered_flag: **CHANGES**
  - N/A
- :bug: **FIXED**
  - https://github.com/mweagle/Sparta/issues/6

## v0.5.2
- :warning: **BREAKING**
  - N/A
- :checkered_flag: **CHANGES**
  - Added [cloudwatchlogs.Event](https://godoc.org/github.com/mweagle/Sparta/aws/cloudwatchlogs#Event) to support unmarshaling CloudWatchLogs data

## v0.5.1
- :warning: **BREAKING**
  - N/A
- :checkered_flag: **CHANGES**
  - Added [LambdaAWSInfo.URLPath](https://godoc.org/github.com/mweagle/Sparta#LambdaAWSInfo.URLPath) to enable _localhost_ testing
    - See <i>explore_test.go</i> for example
- :bug: **FIXED**
  - https://github.com/mweagle/Sparta/issues/8

## v0.5.0
- :warning: **BREAKING**
  - N/A
- :checkered_flag: **CHANGES**
  - Added [sparta.CloudWatchLogsPermission](https://godoc.org/github.com/mweagle/Sparta#CloudWatchLogsPermission) type to support lambda invocation in response to log events.
  - Fixed latent bug on Windows where temporary archives weren't properly deleted
  - The `GO15VENDOREXPERIMENT=1` environment variable for cross compilation is now inherited from the current session.
    - Sparta previously always added it to the environment variables during compilation.
  - Hooked AWS SDK logger so that Sparta `--level debug` log level includes AWS SDK status
    - Also include `debug` level message listing AWS SDK version for diagnostic info
  - Log output includes lambda deployment [package size](http://docs.aws.amazon.com/lambda/latest/dg/limits.html)

## v0.4.0
- :warning: **BREAKING**
  - Change `sparta.Discovery()` return type from `map[string]interface{}` to `sparta.DiscoveryInfo`.
      - This type provides first class access to service-scoped and `DependsOn`-related resource information
- :checkered_flag: **CHANGES**
  - N/A

## v0.3.0
- :warning: **BREAKING**
  - Enforce that a single **Go** function cannot be associated with more than 1 `sparta.LamddaAWSInfo` struct.  
    - This was done so that `sparta.Discovery` can reliably use the enclosing **Go** function name for discovery.
  - Enforce that a non-nil `*sparta.API` value provided to `sparta.Main()` includes a non-empty set of resources and methods
- :checkered_flag: **CHANGES**
 type
    - This type can be used to enable [CloudWatch Events](https://aws.amazon.com/blogs/aws/new-cloudwatch-events-track-and-respond-to-changes-to-your-aws-resources/)
    - See the [SpartaApplication](https://github.com/mweagle/SpartaApplication/blob/master/application.go#L381) example app for a sample usage.
  - `sparta.Discovery` now returns the following CloudFormation [Pseudo Parameters](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/pseudo-parameter-reference.html):
    - _StackName_
    - _StackID_
    - _Region_
  - Upgrade to Mermaid [0.5.7](https://github.com/knsv/mermaid/releases/tag/0.5.7) to fix `describe` rendering failure on Chrome.

## v0.2.0
- :warning: **BREAKING**
  - Changed `NewRequest` to `NewLambdaRequest` to support mock API gateway requests being made in `explore` mode
  - `TemplateDecorator` signature changed to support [go-cloudformation](https://github.com/crewjam/go-cloudformation) representation of the CloudFormation JSON template.
    - /ht @crewjam for https://github.com/crewjam/go-cloudformation
  - Use `sparta.EventSourceMapping` rather than [aws.CreateEventSourceMappingInput](http://docs.aws.amazon.com/sdk-for-go/api/service/lambda.html#type-CreateEventSourceMappingInput) type for `LambdaAWSInfo.EventSourceMappings` slice
  - Add dependency on [crewjam/go-cloudformation](https://github.com/crewjam/go-cloudformation) for CloudFormation template creation
    - /ht @crewjam for the great library
  - CloudWatch log output no longer automatically uppercases all first order child key names.

- :checkered_flag: **CHANGES**
  - :boom: Add `LambdaAWSInfo.DependsOn` slice
    -  Lambda functions can now declare explicit dependencies on resources added via a `TemplateDecorator` function
    - The `DependsOn` value should be the dependency's logical resource name.  Eg, the value returned from `CloudFormationResourceName(...)`.
  - :boom: Add `sparta.Discovery()` function
    - To be called from a **Go** lambda function (Eg, `func echoEvent(*json.RawMessage, *LambdaContext, http.ResponseWriter, *logrus.Logger)`), it returns the Outputs (both [Fn::Att](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference-getatt.html) and [Ref](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference-ref.html) ) values of dynamically generated CloudFormation resources that are declared as explicit `DependsOn` of the current function.
    - Sample output return value:

        ```json
        {
          "SESMessageStoreBucketa622fdfda5789d596c08c79124f12b978b3da772": {
            "DomainName": "spartaapplication-sesmessagestorebucketa622fdfda5-1rhh9ckj38gt4.s3.amazonaws.com",
            "Ref": "spartaapplication-sesmessagestorebucketa622fdfda5-1rhh9ckj38gt4",
            "Tags": [
              {
                "Key": "sparta:logicalBucketName",
                "Value": "Special"
              }
            ],
            "Type": "AWS::S3::Bucket",
            "WebsiteURL": "http://spartaapplication-sesmessagestorebucketa622fdfda5-1rhh9ckj38gt4.s3-website-us-west-2.amazonaws.com"
          },
          "golangFunc": "main.echoSESEvent"
        }
        ```

        - See the [SES EventSource docs](http://gosparta.io/docs/eventsources/ses/) for more information.
  - Added `TS` (UTC TimeStamp) field to startup message
  - Improved stack provisioning performance
  - Fixed latent issue where CloudFormation template wasn't deleted from S3 on stack provisioning failure.
  - Refactor AWS runtime requirements into `lambdaBinary` build tag scope to support Windows builds.
  - Add `SESPermission` type to support triggering Lambda functions in response to inbound email
    - See _doc_sespermission_test.go_ for an example
    - Storing the message body to S3 is done by assigning the `MessageBodyStorage` field.
  - Add `NewAPIGatewayRequest` to support _localhost_ API Gateway mock requests

## v0.1.5
- :warning: **BREAKING**
  - N/A
- :checkered_flag: **CHANGES**
  - Add [S3 Object Expiration](http://docs.aws.amazon.com/AmazonS3/latest/dev/how-to-set-lifecycle-configuration-intro.html) warning message if the target bucket doesn't specify one.
  - Replace internal CloudFormation polling loop with [WaitUntilStackCreateComplete](https://godoc.org/github.com/aws/aws-sdk-go/service/cloudformation#CloudFormation.WaitUntilStackCreateComplete) and [WaitUntilStackUpdateComplete](https://godoc.org/github.com/aws/aws-sdk-go/service/cloudformation#CloudFormation.WaitUntilStackUpdateComplete)

## v0.1.4
- :warning: **BREAKING**
  - N/A
- :checkered_flag: **CHANGES**
  - Reduce deployed binary size by excluding Sparta embedded resources from deployed binary via build tags.

## v0.1.3
- :warning: **BREAKING**
  - API Gateway responses are only transformed into a standard format in the case of a go lambda function returning an HTTP status code >= 400
    - Previously all responses were wrapped which prevented integration with other services.
- :checkered_flag: **CHANGES**
  - Default [integration mappings](http://docs.aws.amazon.com/apigateway/latest/developerguide/api-gateway-mapping-template-reference.html) now defined for:
    * _application/json_
    * _text/plain_
    * _application/x-www-form-urlencoded_
    * _multipart/form-data_
    - Depending on the content-type, the **Body** value of the incoming event will either be a `string` or a `json.RawMessage` type.
  - CloudWatch log files support spawned golang binary JSON formatted logfiles
  - CloudWatch log output includes environment.  Sample:

    ```JSON
      {
          "AWS_SDK": "2.2.25",
          "NODE_JS": "v0.10.36",
          "OS": {
              "PLATFORM": "linux",
              "RELEASE": "3.14.48-33.39.amzn1.x86_64",
              "TYPE": "Linux",
              "UPTIME": 4755.330878024
          }
      }
    ```

## v0.1.2
- :warning: **BREAKING**
  - N/A
- :checkered_flag: **CHANGES**
  - Added `explore.NewRequest` to support _localhost_ testing of lambda functions.  
    - Clients can supply optional **event** data similar to the AWS Console feature.
    - See [explore_test](https://github.com/mweagle/Sparta/blob/master/explore_test.go) for an example.

## v0.1.1
- :warning: **BREAKING**
  - `sparta.Main()` signature changed to accept optional `S3Site` pointer
- :checkered_flag: **CHANGES**
  - Updated `describe` CSS font styles to eliminate clipping
  - Support `{Ref: 'MyDynamicResource'}` for _SourceArn_ values.  Example:

    ```javascript
    lambdaFn.Permissions = append(lambdaFn.Permissions, sparta.SNSPermission{
  		BasePermission: sparta.BasePermission{
  			SourceArn: sparta.ArbitraryJSONObject{"Ref": snsTopicName},
  		},
  	})
    ```

    - Where _snsTopicName_ is a CloudFormation resource name representing a resource added to the template via a [TemplateDecorator](https://godoc.org/github.com/mweagle/Sparta#TemplateDecorator).
  - Add CloudWatch metrics to help track [container reuse](https://aws.amazon.com/blogs/compute/container-reuse-in-lambda/).
    - Metrics are published to **Sparta/<SERVICE_NAME>** namespace.
    - MetricNames: `ProcessCreated`, `ProcessReused`, `ProcessTerminated`.

## v0.1.0
- :warning: **BREAKING**
  - `sparta.Main()` signature changed to accept optional `S3Site` pointer
- :checkered_flag: **CHANGES**
  - Added `S3Site` type and optional static resource provisioning as part of `provision`
    - See the [SpartaHTML](https://github.com/mweagle/SpartaHTML) application for a complete example
  - Added `API.CORSEnabled` option (defaults to _false_).  
    - If defined, all APIGateway methods will have [CORS Enabled](http://docs.aws.amazon.com/apigateway/latest/developerguide/how-to-cors.html).
  - Update logging to use structured fields rather than variadic, concatenation
  - Reimplement `explore` command line option.
    - The `explore` command line option creates a _localhost_ server to which requests can be sent for testing.  The POST request body **MUST** be _application/json_, with top level `event` and `context` keys for proper unmarshaling.
  - Expose NewLambdaHTTPHandler() which can be used to generate an _httptest_

## v0.0.7
- :warning: **BREAKING**
  - N/A
- :checkered_flag: **CHANGES**
  - Documentation moved to [gosparta.io](http://gosparta.io)
 compliant value for `go test` integration.
    - Add [context](http://docs.aws.amazon.com/apigateway/latest/developerguide/api-gateway-mapping-template-reference.html) struct to APIGatewayLambdaJSONEvent
    - Default description based on *Go* function name for AWS Lambda if none provided
    - Added [SNS Event](https://github.com/mweagle/Sparta/blob/master/aws/sns/events.go) types for unmarshaling
    - Added [DynamoDB Event](https://github.com/mweagle/Sparta/blob/master/aws/dynamodb/events.go) types for unmarshaling
    - Added [Kinesis Event](https://github.com/mweagle/Sparta/blob/master/aws/kinesis/events.go) types for unmarshaling
    - Fixed latent issue where `IAMRoleDefinition` CloudFormation names would collide if they had the same Permission set.
    - Remove _API Gateway_ view from `describe` if none is defined.


## v0.0.6
- :warning: **BREAKING**
  - Changed:
    - `type LambdaFunction func(*json.RawMessage, *LambdaContext, *http.ResponseWriter, *logrus.Logger)`
      - **TO**
    - `type LambdaFunction func(*json.RawMessage, *LambdaContext, http.ResponseWriter, *logrus.Logger)`
    - See also [FAQ: When should I use a pointer to an interface?](https://golang.org/doc/faq#pointer_to_interface).
- Add _.travis.yml_ for CI support.
- :checkered_flag: **CHANGES**
    - Added [LambdaAWSInfo.Decorator](https://github.com/mweagle/Sparta/blob/master/sparta.go#L603) field (type [TemplateDecorator](https://github.com/mweagle/Sparta/blob/master/sparta.go#L192) ). If defined, the template decorator will be called during CloudFormation template creation and enables a Sparta lambda function to annotate the CloudFormation template with additional Resources or Output entries.
      - See [TestDecorateProvision](https://github.com/mweagle/Sparta/blob/master/provision_test.go#L44) for an example.
    - Improved API Gateway `describe` output.
    - Added [method response](http://docs.aws.amazon.com/apigateway/api-reference/resource/method-response/) support.  
      - The [DefaultMethodResponses](https://godoc.org/github.com/mweagle/Sparta#DefaultMethodResponses) map is used if [Method.Responses](https://godoc.org/github.com/mweagle/Sparta#Method) is empty  (`len(Responses) <= 0`) at provision time.
      - The default response map defines `201` for _POST_ methods, and `200` for all other methods. An API Gateway method may only support a single 2XX status code.
    - Added [integration response](http://docs.aws.amazon.com/apigateway/api-reference/resource/integration-response/) support for to support HTTP status codes defined in [status.go](https://golang.org/src/net/http/status.go).
      - The [DefaultIntegrationResponses](https://godoc.org/github.com/mweagle/Sparta#DefaultIntegrationResponses) map is used if [Integration.Responses](https://godoc.org/github.com/mweagle/Sparta#Integration) is empty  (`len(Responses) <= 0`) at provision time.
      - The mapping uses regular expressions based on the standard _golang_ [HTTP StatusText](https://golang.org/src/net/http/status.go) values.
    - Added `SpartaHome` and `SpartaVersion` template [outputs](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/outputs-section-structure.html).

## v0.0.5
- :warning: **BREAKING**
  - Changed `Sparta.Main()` signature to accept API pointer as fourth argument.  Parameter is optional.
- :checkered_flag: **CHANGES**
  - Preliminary support for API Gateway provisioning
    - See API type for more information.
  - `describe` output includes:
    - Dynamically generated CloudFormation Template
    - API Gateway json
    - Lambda implementation of `CustomResources` for push source configuration promoted from inline [ZipFile](http://docs.aws.amazon.com/lambda/latest/dg/API_FunctionCode.html) JS code to external JS files that are proxied via _index.js_ exports.
    - [Fixed latent bug](https://github.com/mweagle/Sparta/commit/684b48eb0c2356ba332eee6054f4d57fc48e1419) where remote push source registrations were deleted during stack updates.

## v0.0.3
- :warning: **BREAKING**
  - Changed `LambdaEvent` type to `json.RawMessage`
  - Changed  [AddPermissionInput](http://docs.aws.amazon.com/sdk-for-go/api/service/lambda.html#type-AddPermissionInput) type to _sparta_ types:
    - `LambdaPermission`
    - `S3Permission`
    - `SNSPermission`
- :checkered_flag: **CHANGES**
  - `sparta.NewLambda(...)` supports either `string` or `sparta.IAMRoleDefinition` types for the IAM role execution value
    - `sparta.IAMRoleDefinition` types implicitly create an [IAM::Role](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-iam-role.html) resource as part of the stack
    - `string` values refer to pre-existing IAM rolenames
  - `S3Permission` type
    - `S3Permission` types denotes an S3 [event source](http://docs.aws.amazon.com/lambda/latest/dg/intro-core-components.html#intro-core-components-event-sources) that should be automatically configured as part of the service definition.
    - S3's [LambdaConfiguration](http://docs.aws.amazon.com/sdk-for-go/api/service/s3.html#type-LambdaFunctionConfiguration) is managed by a [Lambda custom resource](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/template-custom-resources-lambda.html) dynamically generated as part of in the [CloudFormation template](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/template-custom-resources.html).
    - The subscription management resource is inline NodeJS code and leverages the [cfn-response](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/walkthrough-custom-resources-lambda-cross-stack-ref.html) module.
  - `SNSPermission` type
    - `SNSPermission` types denote an SNS topic that should should send events to the target Lambda function
    - An SNS Topic's [subscriber list](http://docs.aws.amazon.com/AWSJavaScriptSDK/latest/AWS/SNS.html#subscribe-property) is managed by a [Lambda custom resource](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/template-custom-resources-lambda.html) dynamically generated as part of in the [CloudFormation template](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/template-custom-resources.html).
   - The subscription management resource is inline NodeJS code and leverages the [cfn-response](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/walkthrough-custom-resources-lambda-cross-stack-ref.html) module.
  - `LambdaPermission` type
    - These denote Lambda Permissions whose event source subscriptions should **NOT** be managed by the service definition.
  - Improved `describe` output CSS and layout
    - Describe now includes push/pull Lambda event sources
  - Fixed latent bug where Lambda functions didn't have CloudFormation::Log privileges

## v0.0.2
  - Update describe command to use [mermaid](https://github.com/knsv/mermaid) for resource dependency tree
    - Previously used [vis.js](http://visjs.org/#)

## v0.0.1
  - Initial release
