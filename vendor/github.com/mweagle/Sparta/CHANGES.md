# Change Notes

## v0.20.4
- :warning: **BREAKING**
  - Changed `step.NewStateMachine` signature to include _StateMachineName_ as first argument per [Nov 15th, 2017 release](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/ReleaseHistory.html)
- :checkered_flag: **CHANGES**
  * Add `profile` command
    - Profile snapshots are enabled via:
    ```
    sparta.ScheduleProfileLoop(nil, 5*time.Second, 30*time.Second, "heap")
    ```
    - Profile snapshots are published to S3 and are locally aggregated across all lambda instance publishers. To view the ui, run the `profile` Sparta command.
      - For more information, please see [The new pprof user interface - ⭐️](https://rakyll.org/pprof-ui/), [Profiling Go programs with pprof](https://jvns.ca/blog/2017/09/24/profiling-go-with-pprof/), or the [Go blog](https://blog.golang.org/profiling-go-programs)
    - See the [SpartaPProf](https://github.com/mweagle/SpartaPProf) sample for a service that installs profiling hooks.
    - Ensure you have the latest `pprof` UI via _go get -u -v github.com/google/pprof_
    - The standard [profile names](https://golang.org/pkg/runtime/pprof/#Profile) are available, as well as a _cpu_ type implied by a non-zero `time.Duration` supplied as the third parameter to `ScheduleProfileLoop`.
  * Eliminate unnecessary logging in AWS lambda environment
  * Log NodeJS [process.uptime()](https://nodejs.org/api/process.html#process_process_uptime)
- :bug:  **FIXED**
  * Added more constructive message when working directory for `go build` doesn't contain `main` package.

## v0.20.3
- :warning: **BREAKING**
- :checkered_flag: **CHANGES**
- :bug:  **FIXED**
  * Fixed `explore` interactive debugging instructions

## v0.20.2
- :warning: **BREAKING**
- :checkered_flag: **CHANGES**
  - Added support for [Step functions](https://aws.amazon.com/step-functions/faqs/).
    - Step functions are expressed via a combination of: states, `NewStateMachine`, and adding a `StateMachineDecorator` as a [service hook](https://godoc.org/github.com/mweagle/Sparta#ServiceDecoratorHook).
    - See the [SpartaStep](https://github.com/mweagle/SpartaStep) sample for a service that provisions a simple roll die state machine.
  - Usability improvements & enhancements for CLI log output. Text-formatted output now includes cleaner header as in:
    ```
    INFO[0000] ══════════════════════════════════════════════════════════════
    INFO[0000]    _______  ___   ___  _________
    INFO[0000]   / __/ _ \/ _ | / _ \/_  __/ _ |     Version : 0.20.2
    INFO[0000]  _\ \/ ___/ __ |/ , _/ / / / __ |     SHA     : 740028b
    INFO[0000] /___/_/  /_/ |_/_/|_| /_/ /_/ |_|     Go      : go1.9.1
    INFO[0000]
    INFO[0000] ══════════════════════════════════════════════════════════════
    INFO[0000] Service: SpartaStep-mweagle                   LinkFlags= Option=provision UTC="2017-11-01T01:14:31Z"
    INFO[0000] ══════════════════════════════════════════════════════════════
    ```
  - Added [megacheck](https://github.com/dominikh/go-tools/tree/master/cmd/megacheck) to compile pipeline. Fixed issues.
  - Corrected inline Go examples to use proper function references & signatures.
- :bug:  **FIXED**
  * Handle case where multipart forms with empty values weren't handled [https://github.com/mweagle/Sparta/issues/74]

## v0.20.1
- :warning: **BREAKING**
- :checkered_flag: **CHANGES**
  - Add `sparta.LambdaName` to return the reflection-discovered name of an `http.HandleFunc` instance.
- :bug:  **FIXED**
  * Fixed issue with `--describe` not rendering CloudFormation template properly
  * Better handle failures when [posting body](https://github.com/mweagle/Sparta/pull/72) - thanks https://github.com/nylar

## v0.20.0

### :star: Deprecation Notice

The `sparta.LambdaFunc` signature is officially deprecated in favor of `http.HandlerFunc` and will be removed in an upcoming release. See below for more information

- :warning: **BREAKING**
  - Changed `NewLambdaHTTPHandler` to `NewServeMuxLambda`
  - Remove obsolete `InvokeID` from [LambdaContext](https://godoc.org/github.com/mweagle/Sparta#LambdaContext)
  - Changed `codePipelineTrigger` CLI arg name to `codePipelinePackage`
- :checkered_flag: **CHANGES**
  - Eliminated NodeJS cold start `cp & chmod` penalty! :fire:
    - Prior to this release, the NodeJS proxying code would copy the embedded binary to _/tmp_ and add the executable flag prior to actually launching the binary. This had a noticable performance penalty for startup.
    - This release embeds the application or library in a _./bin_ directory with the file permissions set so that there is no additional filesystem overhead on cold-start. h/t to [StackOverflow](https://stackoverflow.com/questions/41651134/cant-run-binary-from-within-python-aws-lambda-function) for the tips.
  - Migrated all IPC calls to [protocolBuffers](https://developers.google.com/protocol-buffers/).
    - Message definitions are in the [proxy](https://github.com/mweagle/Sparta/tree/master/proxy) directory.
  - The client-side log level (eg: `--level debug`) is carried into the AWS Lambda Code package.
    - Provisioning a service with `--level debug` will log everything at `logger.Debug` level and higher **including all AWS API** calls made both at `provision` and Lambda execution time.
    - Help resolve "Works on My Stack" syndrome.
  - HTTP handler `panic` events are now recovered and the traceback logged for both NodeJS and `cgo` deployments
  - Introduced `sparta.HandleAWSLambda`
    - `sparta.HandleAWSLambda` accepts standard `http.RequestFunc` signatures as in:
      ```
      func helloWorld(w http.ResponseWriter, r *http.Request) {
        ...
      }

      lambdaFn := sparta.HandleAWSLambda("Hello HTTP World",
		    http.HandlerFunc(helloWorld),
		    sparta.IAMRoleDefinition{})
      ```
    - This allows you to [chain middleware](https://github.com/justinas/alice) for a lambda function as if it were a standard HTTP handler. Say, for instance: [X-Ray](https://github.com/aws/aws-xray-sdk-go).
    - The legacy [sparta.LambdaFunction](https://godoc.org/github.com/mweagle/Sparta#LambdaFunction) is still supported, but marked for deprecation. You will see a log warning as in:
      ```
      WARN[0045] DEPRECATED: sparta.LambdaFunc() signature provided. Please migrate to http.HandlerFunc()
      ```
    - _LambdaContext_ and _*logrus.Logger_ are now available in the [requext.Context()](https://golang.org/pkg/net/http/#Request.Context) via:
      * `sparta.ContextKeyLogger` => `*logrus.Logger`
      * `sparta.ContextKeyLambdaContext` => `*sparta.LambdaContext`
    - Example:
      - `loggerVal, loggerValOK := r.Context().Value(sparta.ContextKeyLogger).(*logrus.Logger)`
  - Added support for [CodePipeline](https://aws.amazon.com/about-aws/whats-new/2016/11/aws-codepipeline-introduces-aws-cloudformation-deployment-action/)
    - See the [SpartaCodePipeline](https://github.com/mweagle/SpartaCodePipeline) project for a complete example and the related [post](https://medium.com/@mweagle/serverless-serverfull-and-weaving-pipelines-c9f83eec9227).
  - Upgraded NodeJS to [nodejs6.10](http://docs.aws.amazon.com/lambda/latest/dg/API_CreateFunction.html#SSS-CreateFunction-request-Runtime) runtime
  - Parity between NodeJS and Python/`cgo` startup output
  - Both NodeJS and `cgo` based Sparta applications now log equivalent system information.
      - Example:
      ```
      {
        "level": "info",
        "msg": "SystemInfo",
        "systemInfo": {
            "sysinfo": {
                "version": "0.9.1",
                "timestamp": "2017-09-16T17:07:34.491807588Z"
            },
            "node": {
                "hostname": "ip-10-25-51-97",
                "machineid": "0046d1358d2346adbf8851e664b30d25",
                "hypervisor": "xenhvm",
                "timezone": "UTC"
            },
            "os": {
                "name": "Amazon Linux AMI 2017.03",
                "vendor": "amzn",
                "version": "2017.03",
                "architecture": "amd64"
            },
            "kernel": {
                "release": "4.9.43-17.38.amzn1.x86_64",
                "version": "#1 SMP Thu Aug 17 00:20:39 UTC 2017",
                "architecture": "x86_64"
            },
            "product": {},
            "board": {},
            "chassis": {},
            "bios": {},
            "cpu": {
                "vendor": "GenuineIntel",
                "model": "Intel(R) Xeon(R) CPU E5-2680 v2 @ 2.80GHz",
                "cache": 25600,
                "threads": 2
            },
            "memory": {}
        },
        "time": "2017-09-16T17:07:34Z"
    }
    ```
- :bug: **FIXED**
  - There were more than a few

## v0.13.2
- :warning: **BREAKING**
- :checkered_flag: **CHANGES**
  - Changed how Lambda [FunctionName](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-lambda-function.html#cfn-lambda-function-functionname) values are defined so that function name uniqueness is preserved for free, imported free, and struct-defined functions
- :bug: **FIXED**

## v0.13.1
- :warning: **BREAKING**
- :checkered_flag: **CHANGES**
  - Changed how Lambda [FunctionName](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-lambda-function.html#cfn-lambda-function-functionname) values are defined so that same-named functions provisioned across multiple stacks remain unique. This is done by prefixing the function name with the CloudFormation StackName.
  - Cleaned up S3 upload log statements to prefer relative paths iff applicable
- :bug: **FIXED**
  - [Cloudformation lambda function name validation error](https://github.com/mweagle/Sparta/issues/63)
  - [Python error with cgo lambda ](https://github.com/mweagle/Sparta/issues/64)

## v0.13.0
- :warning: **BREAKING**
  - Removed `sparta.NewNamedLambda`. Stable, user-defined function names can be supplied via the [SpartaOptions.Name](https://godoc.org/github.com/mweagle/Sparta#SpartaOptions) field.
- :checkered_flag: **CHANGES**
  - [CloudWatch Dashboard Support!](http://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/CloudWatch_Dashboards.html)
    - You can provision a CloudWatch dashboard that provides a single overview and link portal for your Lambda-based service. Use the new `sparta.DashboardDecorator` function to automatically create a dashboard. This leverages the existing [WorkflowHooks](https://godoc.org/github.com/mweagle/Sparta#WorkflowHooks) functionality.
    - Example:
    ```
    // Setup the DashboardDecorator lambda hook
    workflowHooks := &sparta.WorkflowHooks{
      ServiceDecorator: sparta.DashboardDecorator(lambdaFunctions, 60),
    }
    ```
    - Where the `60` value is the CloudWatch time series [period](http://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/cloudwatch_concepts.html).
    - The CloudWatch Dashboard URL will be included in your stack's Outputs as in:
    ```
    INFO[0064] Stack output                                  Description="CloudWatch Dashboard URL" Key=CloudWatchDashboardURL Value="https://us-west-2.console.aws.amazon.com/cloudwatch/home?region=us-west-2#dashboards:name=SpartaXRay-mweagle"
    ```
    - *Example*: <div align="center"><img src="https://raw.githubusercontent.com/mweagle/Sparta/master/site/0.13.0/dashboard.jpg" />
    </div>

    - For more info, see the [AWS Blog Post](https://aws.amazon.com/blogs/aws/new-api-cloudformation-support-for-amazon-cloudwatch-dashboards/)
    - The [SpartaXRay](https://github.com/mweagle/SpartaXRay) sample application has additional code samples.
  - [XRay](http://docs.aws.amazon.com/xray/latest/devguide/xray-services-lambda.html) support added
    - added `LambdaFunctionOptions.TracingConfig` field to [LambdaFunctionOptions](https://godoc.org/github.com/mweagle/Sparta#LambdaFunctionOptions)
    - added XRay IAM privileges to default IAM role settings:
      - _xray:PutTraceSegments_
      - _xray:PutTelemetryRecords_
    - See [AWS blog](https://aws.amazon.com/blogs/aws/aws-lambda-support-for-aws-x-ray/) for more information
  - added [LambdaFunctionOptions.Tags](https://godoc.org/github.com/mweagle/Sparta#LambdaFunctionOptions) to support tagging AWS Lambda functions
  - added _SpartaGitHash_ output to both CLI and CloudWatch Dashboard output. This is in addition to the _SpartaVersion_ value (which I occassionally have failed to update).
- :bug: **FIXED**
  - Fixed latent issue where `SpartaOptions.Name` field wasn't consistently used for function names.

## v0.12.1
- :warning: **BREAKING**
- :checkered_flag: **CHANGES**
  - added _Sparta/aws/cloudformation.UserScopedStackName()_ to generate username-suffixed CloudFormation StackNames
- :bug: **FIXED**

## v0.12.0
- :warning: **BREAKING**
  - Replaced all https://github.com/crewjam/go-cloudformation references with https://github.com/mweagle/go-cloudformation references
    - This is mostly internal facing, but impacts advanced usage via [ServiceDecoratorHook](https://godoc.org/github.com/mweagle/Sparta#ServiceDecoratorHook) users. Clients may
    need to update the types used to create [alternative topologies](http://gosparta.io/docs/alternative_topologies/).
- :checkered_flag: **CHANGES**
- :bug: **FIXED**
  - Fixed latent issue where CGO-enabled services that reference `cgo.NewSession()` would not build properly
  - Fixed latent issue where S3 backed sites (eg: [SpartaHugo](https://github.com/mweagle/SpartaHugo)) would not refresh on update.
  - https://github.com/mweagle/Sparta/issues/55

## v0.11.2
- :warning: **BREAKING**
- :checkered_flag: **CHANGES**
  - Added `--inplace/-c` command line option to support safe, concurrent updating of Lambda code packages
    - If enabled *AND* the stack update [changeset](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/using-cfn-updating-stacks-changesets.html) reports _only_ modifications to Lambda functions, then Sparta will use the AWS Lambda API to [update the function code](http://docs.aws.amazon.com/sdk-for-go/api/service/lambda/#Lambda.UpdateFunctionCode).
    - If enabled *AND* additional mutations are reported, you'll see an error as in:
    ```
    ERRO[0022] Unsupported in-place operations detected:
      Add for IAMRole9fd267df3a3d0a144ae11a64c7fb9b7ffff3fb6c (ResourceType: AWS::IAM::Role),
      Add for mainhelloWorld2Lambda32fcf388f6b20e86feb93e990fa8decc5b3f9095 (ResourceType: AWS::Lambda::Function)
    ```
  - Prefer [NewRecorder](https://golang.org/pkg/net/http/httptest/#NewRecorder) to internal type for CGO marshalling
  - Added `--format/-f` command line flag `[text, txt, json]` to specify logfile output format. Default is `text`.
    - See [logrus.Formatters](https://github.com/sirupsen/logrus#formatters)
- :bug: **FIXED**
  - https://github.com/mweagle/Sparta/issues/45

## v0.11.1
- :warning: **BREAKING**
- :checkered_flag: **CHANGES**
  - Support Go 1.8 newly optional _GOPATH_ environment variable
  - Python proxied `cgo` builds now preserve the transformed source in the _./sparta_ scratch space directory.
  - Sparta assigned AWS Lambda function names now strip the leading SCM prefix. Example:
  ```bash
  github.com/mweagle/SpartaPython.HelloWorld
  ```
  becomes:
  ```bash
  mweagle/SpartaPython.HelloWorld
  ```
  - Upgrade to Mermaid [7.0.0](https://github.com/knsv/mermaid/releases/tag/7.0.0)
  - Use stable _PolicyName_ in `IAM::Role` definitions to minimize CloudFormation resource update churn
- :bug: **FIXED**
  - Fixed latent bug where S3 bucket version check didn't respect `--noop` mode.
  - Fixed latent `cgo` bug where command line arguments weren't properly parsed

## v0.11.0
- :warning: **BREAKING**
- :checkered_flag: **CHANGES**
  - :tada: Python CGO support added. See the https://github.com/mweagle/SpartaPython project for example usage!
    - In preliminary testing, the Python CGO package provides significant cold start and hot-execution performance benefits.
  - Migrated dependency management to [dep](https://github.com/golang/dep)
- :bug: **FIXED**
  - Fixed latent bug where DynamoDB EventSource mappings ResourceARNs weren't properly serialized.
  - Fixed latent bug where code pushed to S3 version-enabled buckets didn't use the latest `VersionID` in the AWS [Lambda Code](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-function-code.html) value.

## v0.10.0

- :warning: **BREAKING**
  - `describe` option now requires `-b/--s3Bucket` argument
  - Changed signature of `aws/s3/CreateS3RollbackFunc` to accept full S3 URL, including `versionId` query param
  - Signatures for `sparta.Provision` and `sparta.Discover` updated with new arguments
- :checkered_flag: **CHANGES**
  - Add `-p/--codePipelineTrigger` command line option to generate CodePipeline deployment package
  - Add `sparta.RegisterCodePipelineEnvironment` to define environment variables in support of [CloudFormation Deployments](https://aws.amazon.com/about-aws/whats-new/2016/11/aws-codepipeline-introduces-aws-cloudformation-deployment-action/). Example:
  ```golang
  func init() {
    sparta.RegisterCodePipelineEnvironment("test", map[string]string{
      "MESSAGE": "Hello Test!",
    })
    sparta.RegisterCodePipelineEnvironment("production", map[string]string{
      "MESSAGE": "Hello Production!",
    })
  }
  ```
  - Add support for `Environment` and `KmsKeyArn` properties to [LambdaFunctionOptions](https://godoc.org/github.com/mweagle/Sparta#LambdaFunctionOptions).  See [AWS](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-lambda-function.html) for more information.
 - Move all build artifacts to _./sparta_ directory
  - `-n/--noop` argument orphans S3 artifacts in _./sparta_ directory
  - Add support for S3 version policy enabled buckets
    - Artifacts pushed to S3 version-enabled buckets now use stable object keys. Rollback functions target specific versions if available.
  - Cleanup log statements
  - Add `sparta/aws/session.NewSessionWithLevel()` to support [AWS LogLevel](http://docs.aws.amazon.com/sdk-for-go/api/aws/#LogLevelType) parameter
- :bug: **FIXED**
  - https://github.com/mweagle/Sparta/issues/34
  - https://github.com/mweagle/Sparta/issues/37
  - https://github.com/mweagle/Sparta/issues/38

## v0.9.3

- :warning: **BREAKING**
- :checkered_flag: **CHANGES**
  - Added [LambdaFunctionOptions.SpartaOptions](https://godoc.org/github.com/mweagle/Sparta#SpartaOptions) struct
    - The primary use case is to support programmatically generated lambda functions that must be disambiguated by their Sparta name. Sparta defaults to reflection based function name identification.
  - Added `--ldflags` support to support lightweight [dynamic string variables](https://golang.org/cmd/link/)
    - Usage:
      `go run main.go provision --level info --s3Bucket $(S3_BUCKET) --ldflags "-X main.dynamicValue=SampleValue"`

- :bug: **FIXED**

## v0.9.2

- :warning: **BREAKING**
- :checkered_flag: **CHANGES**
  - Move Sparta-related provisioning values from stack [Outputs](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/outputs-section-structure.html) to [Tags](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-console-add-tags.html).
  - Add support for go [BuildTags](https://golang.org/pkg/go/build/) to support environment settings.
  - Added [Sparta/aws/cloudformation](https://godoc.org/github.com/mweagle/Sparta/aws/cloudformation) functions to support stack creation.
  - Added [Sparta/aws/s3](https://godoc.org/github.com/mweagle/Sparta/aws/cloudformation) functions to encapsulate common S3 operations.
  - Added  [Sparta/zip](https://godoc.org/github.com/mweagle/Sparta/zip) functions to expose common ZIP related functions.
  - Legibility enhancements for `describe` output
  - `sparta.CloudFormationResourceName` proxies to `github.com/mweagle/Sparta/aws/cloudformation.CloudFormationResourceName`. The `sparta` package function is *deprecated* and will be removed in a subsequent release.
- :bug: **FIXED**
  - Fixed latent bug in `github.com/mweagle/Sparta/zip.AddToZip` where the supplied ZipWriter was incorrectly closed on function exit.
  - Fixed latent parsing _userdata_ input
  - Fixed latent issue where empty [ChangeSets](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/using-cfn-updating-stacks-changesets-execute.html) were applied rather than deleted.

## v0.9.1

- :warning: **BREAKING**
- :checkered_flag: **CHANGES**
  - Improved `describe` output. Includes APIGateway resources and more consistent UI.
  - Additive changes to [WorkflowHooks](https://godoc.org/github.com/mweagle/Sparta#WorkflowHooks)
    - `Context` property to set the initial context for Workflow hook execution
    - [ServiceDecorator](https://godoc.org/github.com/mweagle/Sparta#ServiceDecorator) type  to define service-scoped AWS resources. Previously, template decoration was bound to specific Lambda functions.
  - Published related [SpartaVault](https://github.com/mweagle/SpartaVault): use AWS KMS to encrypt secrets as Go variables. See the [KMS Docs](http://docs.aws.amazon.com/kms/latest/developerguide/workflow.html) for information.
  - Add Godeps support
- :bug: **FIXED**
  - Fixed latent bug when adding custom resources to the ZIP archive via [ArchiveHook](https://godoc.org/github.com/mweagle/Sparta#ArchiveHook). ArchiveHook is now called after core Sparta assets are injected into archive.

## v0.9.0

- :warning: **BREAKING**
  - `NewMethod` and `NewAuthorizedMethod` for APIGateway definitions have been changed to include new, final parameter that marks the _default_ integration response code.
    - Prior to this change, Sparta would automatically use `http.StatusOK` for all non-POST requests, and `http.StatusCreated` for POST requests. The change allows you to control whitelisted headers to be returned through APIGateway as in:

    ```golang
    // API response struct
    type helloWorldResponse struct {
      Location string `json:"location"`
      Body     string `json:"body"`
    }
    //
    // Promote the location key value to an HTTP header
    //
    apiGWMethod, _ := apiGatewayResource.NewMethod("GET", http.StatusOK)
    apiGWMethod.Responses[http.StatusOK].Parameters = map[string]bool{
      "method.response.header.Location": true,
    }
    apiGWMethod.Integration.Responses[http.StatusOK].Parameters["method.response.header.Location"] = "integration.response.body.location"
    ```

- :checkered_flag: **CHANGES**
  - (@sdbeard) Added [sparta.NewNamedLambda](https://godoc.org/github.com/mweagle/Sparta#NewNamedLambda) that allows you to set stable AWS Lambda [FunctionNames](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-lambda-function.html#cfn-lambda-function-functionname)
  - Added [spartaCF.AddAutoIncrementingLambdaVersionResource](https://godoc.org/github.com/mweagle/Sparta/aws/cloudformation#AddAutoIncrementingLambdaVersionResource) to support Lambda function versions.  Should be called from a TemplateDecorator. Usage:

    ```golang
    autoIncrementingInfo, autoIncrementingInfoErr := spartaCF.AddAutoIncrementingLambdaVersionResource(serviceName,
      lambdaResourceName,
      cfTemplate,
      logger)
    if nil != autoIncrementingInfoErr {
      return autoIncrementingInfoErr
    }
    ```

  - Added new [CloudWatch Metrics](http://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/CW_Support_For_AWS.html.html#cfn-lambda-function-functionname) for lambda execution
  - Removed all NodeJS shim `dependencies` from _./resources/provision/package.json_
  - Added utility CloudFormation script _./aws/cloudformation/cli/describe.go_ which produces a JSON serialization of a [DescribeStacksOutput](https://godoc.org/github.com/aws/aws-sdk-go/service/cloudformation#DescribeStacksOutput) struct for build-time discovery of cluster-scoped resources.
  - Relaxed constraint that an API GW resource is bound to single Sparta lambda function. You can now register per-HTTP method name lambda functions for the same API GW resource.
  - Added [Contributors](https://github.com/mweagle/Sparta#contributors) section to README
- :bug: **FIXED**
  - https://github.com/mweagle/Sparta/issues/19
  - https://github.com/mweagle/Sparta/issues/16
  - https://github.com/mweagle/Sparta/issues/15

## v0.8.0

- :warning: **BREAKING**
  - `TemplateDecorator` signature changed to include `context map[string]interface{}` to support sharing state across `WorkflowHooks` (below).
- :checkered_flag: **CHANGES**
  - Add `SpartaBuildID` stack output with build ID
  - `WorkflowHooks`
    - WorkflowHooks enable an application to customize the ZIP archive used as the AWS Lambda target rather than needing to embed resources inside their Go binary
    - They may also be used for Docker-based mixed topologies. See
  - Add optional `-i/--buildID` parameter for `provision`.
    - The parameter will be added to the stack outputs
    - A random value will be used if non is provided on the command line
  - Artifacts posted to S3 are now scoped by `serviceName`
  - Add `sparta.MainEx` for non-breaking signature extension
- :bug: **FIXED**
  - (@sdbeard) Fixed latent bug in Kinesis event source subscriptions that caused `ValidationError`s during provisioning:
    ```bash
    ERRO[0028] ValidationError: [/Resources/IAMRole3dbc1b4199ad659e6267d25cfd8cc63b4124530d/Type/Policies/0/PolicyDocument/Statement/5/Resource] 'null' values are not allowed in templates
        status code: 400, request id: ed5fae8e-7103-11e6-8d13-b943b498f5a2
    ```
  - Fixed latent bug in [ConvertToTemplateExpression](https://godoc.org/github.com/mweagle/Sparta/aws/cloudformation#ConvertToTemplateExpression) when parsing input with multiple AWS JSON fragments.
  - Fixed latent bug in [sparta.Discover](https://godoc.org/github.com/mweagle/Sparta#Discover) which prevented dependent resources from being discovered at Lambda execution time.
  - Fixed latent bug in [explore.NewAPIGatewayRequest](https://godoc.org/github.com/mweagle/Sparta/explore#NewAPIGatewayRequest) where whitelisted param keynames were unmarshalled to `method.request.TYPE.VALUE` rather than `TYPE`.

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
    - /ht @crewjam for [go-cloudformation](https://github.com/crewjam/go-cloudformation)
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
