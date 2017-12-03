package sparta

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/asaskevich/govalidator"
	"github.com/spf13/cobra"
)

// CommandLineOptions defines the commands available via the Sparta command
// line interface.  Embedding applications can extend existing commands
// and add their own to the `Root` command.  See https://github.com/spf13/cobra
// for more information.
var CommandLineOptions = struct {
	Root      *cobra.Command
	Version   *cobra.Command
	Provision *cobra.Command
	Delete    *cobra.Command
	Execute   *cobra.Command
	Describe  *cobra.Command
	Explore   *cobra.Command
	Profile   *cobra.Command
}{}

/******************************************************************************/
// Global options
type optionsGlobalStruct struct {
	ServiceName        string         `valid:"required"`
	ServiceDescription string         `valid:"-"`
	Noop               bool           `valid:"-"`
	LogLevel           string         `valid:"matches(panic|fatal|error|warn|info|debug)"`
	LogFormat          string         `valid:"matches(txt|text|json)"`
	Logger             *logrus.Logger `valid:"-"`
	Command            string         `valid:"-"`
	BuildTags          string         `valid:"-"`
	LinkerFlags        string         `valid:"-"` // no requirements
}

// OptionsGlobal stores the global command line options
var OptionsGlobal optionsGlobalStruct

/******************************************************************************/
// Provision options
// Ref: http://docs.aws.amazon.com/AmazonS3/latest/dev/BucketRestrictions.html
type optionsProvisionStruct struct {
	S3Bucket        string `valid:"required,matches(\\w+)"`
	BuildID         string `valid:"matches(\\S+)"` // non-whitespace
	PipelineTrigger string `valid:"-"`
	InPlace         bool   `valid:"-"`
}

var optionsProvision optionsProvisionStruct

func provisionBuildID(userSuppliedValue string) (string, error) {
	buildID := userSuppliedValue
	if "" == buildID {
		hash := sha1.New()
		randomBytes := make([]byte, 256)
		_, err := rand.Read(randomBytes)
		if err != nil {
			return "", err
		}
		hash.Write(randomBytes)
		buildID = hex.EncodeToString(hash.Sum(nil))
	}
	return buildID, nil
}

/******************************************************************************/
// Execute options
type optionsExecuteStruct struct {
	Port            int `valid:"-"`
	SignalParentPID int `valid:"-"`
}

var optionsExecute optionsExecuteStruct

/******************************************************************************/
// Describe options
type optionsDescribeStruct struct {
	OutputFile string `valid:"required"`
	S3Bucket   string `valid:"required,matches(\\w+)"`
}

var optionsDescribe optionsDescribeStruct

/******************************************************************************/
// Explore options
type optionsExploreStruct struct {
	Port int `valid:"-"`
}

var optionsExplore optionsExploreStruct

/******************************************************************************/
// Profile options
type optionsProfileStruct struct {
	S3Bucket string `valid:"required,matches(\\w+)"`
	Port     int    `valid:"-"`
}

var optionsProfile optionsProfileStruct

/******************************************************************************/
// Initialization
// Initialize all the Cobra commands and their associated flags
/******************************************************************************/
func init() {
	// Root
	CommandLineOptions.Root = &cobra.Command{
		Use:   path.Base(os.Args[0]),
		Short: "Sparta-powered AWS Lambda microservice",
	}
	CommandLineOptions.Root.PersistentFlags().BoolVarP(&OptionsGlobal.Noop, "noop",
		"n",
		false,
		"Dry-run behavior only (do not perform mutations)")
	CommandLineOptions.Root.PersistentFlags().StringVarP(&OptionsGlobal.LogLevel,
		"level",
		"l",
		"info",
		"Log level [panic, fatal, error, warn, info, debug]")
	CommandLineOptions.Root.PersistentFlags().StringVarP(&OptionsGlobal.LogFormat,
		"format",
		"f",
		"text",
		"Log format [text, json]")
	CommandLineOptions.Root.PersistentFlags().StringVarP(&OptionsGlobal.BuildTags,
		"tags",
		"t",
		"",
		"Optional build tags for conditional compilation")
	// Make sure there's a place to put any linker flags
	CommandLineOptions.Root.PersistentFlags().StringVar(&OptionsGlobal.LinkerFlags,
		"ldflags",
		"",
		"Go linker string definition flags (https://golang.org/cmd/link/)")

	// Version
	CommandLineOptions.Version = &cobra.Command{
		Use:   "version",
		Short: "Sparta framework version",
		Long:  `Displays the Sparta framework version `,
		Run: func(cmd *cobra.Command, args []string) {

		},
	}
	// Provision
	CommandLineOptions.Provision = &cobra.Command{
		Use:   "provision",
		Short: "Provision service",
		Long:  `Provision the service (either create or update) via CloudFormation`,
	}
	CommandLineOptions.Provision.Flags().StringVarP(&optionsProvision.S3Bucket,
		"s3Bucket",
		"s",
		"",
		"S3 Bucket to use for Lambda source")
	CommandLineOptions.Provision.Flags().StringVarP(&optionsProvision.BuildID,
		"buildID",
		"i",
		"",
		"Optional BuildID to use")
	CommandLineOptions.Provision.Flags().StringVarP(&optionsProvision.PipelineTrigger,
		"codePipelinePackage",
		"p",
		"",
		"Name of CodePipelin package that includes cloduformation.json Template and ZIP config files")
	CommandLineOptions.Provision.Flags().BoolVarP(&optionsProvision.InPlace,
		"inplace",
		"c",
		false,
		"If the provision operation results in *only* function updates, bypass CloudFormation")

	// Delete
	CommandLineOptions.Delete = &cobra.Command{
		Use:   "delete",
		Short: "Delete service",
		Long:  `Ensure service is successfully deleted`,
	}

	// Execute
	CommandLineOptions.Execute = &cobra.Command{
		Use:   "execute",
		Short: "Execute",
		Long:  `Startup the localhost HTTP server to handle requests`,
	}
	CommandLineOptions.Execute.Flags().IntVarP(&optionsExecute.Port,
		"port",
		"p",
		9999,
		"Alternative port for HTTP binding (default=9999)")
	CommandLineOptions.Execute.Flags().IntVarP(&optionsExecute.SignalParentPID,
		"signal",
		"s",
		0,
		"Process ID to signal with SIGUSR2 once ready")

	// Describe
	CommandLineOptions.Describe = &cobra.Command{
		Use:   "describe",
		Short: "Describe service",
		Long:  `Produce an HTML report of the service`,
	}
	CommandLineOptions.Describe.Flags().StringVarP(&optionsDescribe.OutputFile,
		"out",
		"o",
		"",
		"Output file for HTML description")
	CommandLineOptions.Describe.Flags().StringVarP(&optionsDescribe.S3Bucket,
		"s3Bucket",
		"s",
		"",
		"S3 Bucket to use for Lambda source")

	// Explore
	CommandLineOptions.Explore = &cobra.Command{
		Use:   "explore",
		Short: "Interactively explore service",
		Long:  `Startup a localhost HTTP server to explore the exported Go functions`,
	}

	CommandLineOptions.Explore.Flags().IntVarP(&optionsExplore.Port,
		"port",
		"p",
		9999,
		"Alternative port for HTTP binding (default=9999)")

	// Profile
	CommandLineOptions.Profile = &cobra.Command{
		Use:   "profile",
		Short: "Interactively examine service pprof output",
		Long:  `Startup a local pprof webserver to interrogate profiles snapshots on S3`,
	}
	CommandLineOptions.Profile.Flags().StringVarP(&optionsProfile.S3Bucket,
		"s3Bucket",
		"s",
		"",
		"S3 Bucket that stores lambda profile snapshots")
	CommandLineOptions.Profile.Flags().IntVarP(&optionsProfile.Port,
		"port",
		"p",
		8080,
		"Alternative port for HTTP binding (default=8080)")
}

// CommandLineOptionsHook allows embedding applications the ability
// to validate caller-defined command line arguments.  Return an error
// if the command line fails.
type CommandLineOptionsHook func(command *cobra.Command) error

// ParseOptions the command line options
func ParseOptions(handler CommandLineOptionsHook) error {
	// First up, create a dummy Root command for the parse...
	var parseCmdRoot = &cobra.Command{
		Use:           CommandLineOptions.Root.Use,
		Short:         CommandLineOptions.Root.Short,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	parseCmdRoot.PersistentFlags().BoolVarP(&OptionsGlobal.Noop, "noop",
		"n",
		false,
		"Dry-run behavior only (do not perform mutations)")
	parseCmdRoot.PersistentFlags().StringVarP(&OptionsGlobal.LogLevel,
		"level",
		"l",
		"info",
		"Log level [panic, fatal, error, warn, info, debug]")
	parseCmdRoot.PersistentFlags().StringVarP(&OptionsGlobal.LogFormat,
		"format",
		"f",
		"text",
		"Log format [text, json]")
	parseCmdRoot.PersistentFlags().StringVarP(&OptionsGlobal.BuildTags,
		"tags",
		"t",
		"",
		"Optional build tags for conditional compilation")

	// Now, for any user-attached commands, add them to the temporary Parse
	// root command.
	for _, eachUserCommand := range CommandLineOptions.Root.Commands() {
		userProxyCmd := &cobra.Command{
			Use:   eachUserCommand.Use,
			Short: eachUserCommand.Short,
		}
		userProxyCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
			_, validateErr := govalidator.ValidateStruct(OptionsGlobal)
			if nil != validateErr {
				return validateErr
			}
			// Format?
			var formatter logrus.Formatter
			switch OptionsGlobal.LogFormat {
			case "text", "txt":
				formatter = &logrus.TextFormatter{}
			case "json":
				formatter = &logrus.JSONFormatter{}
			}
			logger, loggerErr := NewLoggerWithFormatter(OptionsGlobal.LogLevel, formatter)
			if nil != loggerErr {
				return loggerErr
			}
			OptionsGlobal.Logger = logger

			if handler != nil {
				return handler(userProxyCmd)
			}
			return nil
		}
		userProxyCmd.Flags().AddFlagSet(eachUserCommand.Flags())
		parseCmdRoot.AddCommand(userProxyCmd)
	}

	//////////////////////////////////////////////////////////////////////////////
	// Then add the standard Sparta ones...
	spartaCommands := []*cobra.Command{
		CommandLineOptions.Version,
		CommandLineOptions.Provision,
		CommandLineOptions.Delete,
		CommandLineOptions.Execute,
		CommandLineOptions.Describe,
		CommandLineOptions.Explore,
		CommandLineOptions.Profile,
	}
	CommandLineOptions.Version.PreRunE = func(cmd *cobra.Command, args []string) error {
		if handler != nil {
			return handler(CommandLineOptions.Version)
		}
		return nil
	}
	parseCmdRoot.AddCommand(CommandLineOptions.Version)

	CommandLineOptions.Provision.PreRunE = func(cmd *cobra.Command, args []string) error {
		if handler != nil {
			return handler(CommandLineOptions.Provision)
		}
		return nil
	}
	parseCmdRoot.AddCommand(CommandLineOptions.Provision)

	CommandLineOptions.Delete.PreRunE = func(cmd *cobra.Command, args []string) error {
		if handler != nil {
			return handler(CommandLineOptions.Delete)
		}
		return nil
	}
	parseCmdRoot.AddCommand(CommandLineOptions.Delete)

	CommandLineOptions.Execute.PreRunE = func(cmd *cobra.Command, args []string) error {
		if handler != nil {
			return handler(CommandLineOptions.Execute)
		}
		return nil
	}
	parseCmdRoot.AddCommand(CommandLineOptions.Execute)

	CommandLineOptions.Describe.PreRunE = func(cmd *cobra.Command, args []string) error {
		if handler != nil {
			return handler(CommandLineOptions.Describe)
		}
		return nil
	}
	parseCmdRoot.AddCommand(CommandLineOptions.Describe)

	CommandLineOptions.Explore.PreRunE = func(cmd *cobra.Command, args []string) error {
		if handler != nil {
			return handler(CommandLineOptions.Explore)
		}
		return nil
	}
	parseCmdRoot.AddCommand(CommandLineOptions.Explore)

	CommandLineOptions.Profile.PreRunE = func(cmd *cobra.Command, args []string) error {
		if handler != nil {
			return handler(CommandLineOptions.Profile)
		}
		return nil
	}
	parseCmdRoot.AddCommand(CommandLineOptions.Profile)

	// Assign each command an empty RunE func s.t.
	// Cobra doesn't print out the command info
	for _, eachCommand := range parseCmdRoot.Commands() {
		eachCommand.RunE = func(cmd *cobra.Command, args []string) error {
			return nil
		}
	}
	// Intercept the usage command - we'll end up showing this later
	// in Main...If there is an error, we will show help there...
	parseCmdRoot.SetHelpFunc(func(*cobra.Command, []string) {
		// Swallow help here
	})

	// Run it...
	executeErr := parseCmdRoot.Execute()

	// Cleanup the Sparta specific ones
	for _, eachCmd := range spartaCommands {
		eachCmd.RunE = nil
		eachCmd.PreRunE = nil
	}

	if nil != executeErr {
		parseCmdRoot.SetHelpFunc(nil)
		parseCmdRoot.Root().Help()
	}
	return executeErr
}

// Main defines the primary handler for transforming an application into a Sparta package.  The
// serviceName is used to uniquely identify your service within a region and will
// be used for subsequent updates.  For provisioning, ensure that you've
// properly configured AWS credentials for the golang SDK.
// See http://docs.aws.amazon.com/sdk-for-go/api/aws/defaults.html#DefaultChainCredentials-constant
// for more information.
func Main(serviceName string, serviceDescription string, lambdaAWSInfos []*LambdaAWSInfo, api *API, site *S3Site) error {
	return MainEx(serviceName,
		serviceDescription,
		lambdaAWSInfos,
		api,
		site,
		nil,
		false)
}

// MainEx provides an "extended" Main that supports customizing the standard Sparta
// workflow via the `workflowHooks` parameter.
func MainEx(serviceName string,
	serviceDescription string,
	lambdaAWSInfos []*LambdaAWSInfo,
	api *API,
	site *S3Site,
	workflowHooks *WorkflowHooks,
	useCGO bool) error {
	//////////////////////////////////////////////////////////////////////////////
	// cmdRoot defines the root, non-executable command
	CommandLineOptions.Root.Short = fmt.Sprintf("%s - Sparta v.%s powered AWS Lambda Microservice", serviceName, SpartaVersion)
	CommandLineOptions.Root.Long = serviceDescription
	CommandLineOptions.Root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Save the ServiceName in case a custom command wants it
		OptionsGlobal.ServiceName = serviceName
		OptionsGlobal.ServiceDescription = serviceDescription

		_, validateErr := govalidator.ValidateStruct(OptionsGlobal)
		if nil != validateErr {
			return validateErr
		}
		// Format?
		// If we're running in AWS, then pick some sensible defaults
		// per http://docs.aws.amazon.com/lambda/latest/dg/current-supported-versions.html
		runningInLambda := ("" != os.Getenv("LAMBDA_TASK_ROOT"))
		prettyHeader := false
		var formatter logrus.Formatter
		if !runningInLambda {
			switch OptionsGlobal.LogFormat {
			case "text", "txt":
				formatter = &logrus.TextFormatter{}
				prettyHeader = true
			case "json":
				formatter = &logrus.JSONFormatter{}
			}
		} else {
			formatter = &logrus.JSONFormatter{}
		}

		logger, loggerErr := NewLoggerWithFormatter(OptionsGlobal.LogLevel, formatter)
		if nil != loggerErr {
			return loggerErr
		}
		OptionsGlobal.Logger = logger

		welcomeMessage := fmt.Sprintf("Service: %s", serviceName)

		if prettyHeader {
			logger.Info(headerDivider)
			logger.Info(fmt.Sprintf(`   _______  ___   ___  _________ `))
			logger.Info(fmt.Sprintf(`  / __/ _ \/ _ | / _ \/_  __/ _ |     Version : %s`, SpartaVersion))
			logger.Info(fmt.Sprintf(` _\ \/ ___/ __ |/ , _/ / / / __ |     SHA     : %s`, SpartaGitHash[0:7]))
			logger.Info(fmt.Sprintf(`/___/_/  /_/ |_/_/|_| /_/ /_/ |_|     Go      : %s`, runtime.Version()))
			logger.Info(headerDivider)
			logger.WithFields(logrus.Fields{
				"Option":    cmd.Name(),
				"UTC":       (time.Now().UTC().Format(time.RFC3339)),
				"LinkFlags": OptionsGlobal.LinkerFlags,
			}).Info(welcomeMessage)
			logger.Info(headerDivider)
		} else {
			if !runningInLambda {
				logger.Info(headerDivider)
			}
			logger.WithFields(logrus.Fields{
				"Option":        cmd.Name(),
				"SpartaVersion": SpartaVersion,
				"SpartaSHA":     SpartaGitHash[0:7],
				"Go Version":    runtime.Version(),
				"UTC":           (time.Now().UTC().Format(time.RFC3339)),
				"LinkFlags":     OptionsGlobal.LinkerFlags,
			}).Info(welcomeMessage)
			if !runningInLambda {
				logger.Info(headerDivider)
			}
		}
		return nil
	}

	//////////////////////////////////////////////////////////////////////////////
	// Version
	CommandLineOptions.Root.AddCommand(CommandLineOptions.Version)

	//////////////////////////////////////////////////////////////////////////////
	// Provision
	CommandLineOptions.Provision.PreRunE = func(cmd *cobra.Command, args []string) error {
		validationResults, validateErr := govalidator.ValidateStruct(optionsProvision)

		OptionsGlobal.Logger.WithFields(logrus.Fields{
			"validationResults": validationResults,
			"validateErr":       validateErr,
			"optionsProvision":  optionsProvision,
		}).Debug("Provision validation results")
		return validateErr
	}

	if nil == CommandLineOptions.Provision.RunE {
		CommandLineOptions.Provision.RunE = func(cmd *cobra.Command, args []string) error {
			buildID, buildIDErr := provisionBuildID(optionsProvision.BuildID)
			if nil != buildIDErr {
				return buildIDErr
			}
			return Provision(OptionsGlobal.Noop,
				serviceName,
				serviceDescription,
				lambdaAWSInfos,
				api,
				site,
				optionsProvision.S3Bucket,
				useCGO,
				optionsProvision.InPlace,
				buildID,
				optionsProvision.PipelineTrigger,
				OptionsGlobal.BuildTags,
				OptionsGlobal.LinkerFlags,
				nil,
				workflowHooks,
				OptionsGlobal.Logger)
		}
	}
	CommandLineOptions.Root.AddCommand(CommandLineOptions.Provision)

	//////////////////////////////////////////////////////////////////////////////
	// Delete
	CommandLineOptions.Delete.RunE = func(cmd *cobra.Command, args []string) error {
		return Delete(serviceName, OptionsGlobal.Logger)
	}

	CommandLineOptions.Root.AddCommand(CommandLineOptions.Delete)

	//////////////////////////////////////////////////////////////////////////////
	// Execute
	if nil == CommandLineOptions.Execute.RunE {
		CommandLineOptions.Execute.RunE = func(cmd *cobra.Command, args []string) error {
			_, validateErr := govalidator.ValidateStruct(optionsExecute)
			if nil != validateErr {
				return validateErr
			}

			OptionsGlobal.Logger.Formatter = new(logrus.JSONFormatter)
			// Ensure the discovery service is initialized
			initializeDiscovery(serviceName, lambdaAWSInfos, OptionsGlobal.Logger)

			return Execute(lambdaAWSInfos,
				optionsExecute.Port,
				optionsExecute.SignalParentPID,
				OptionsGlobal.Logger)
		}
	}
	CommandLineOptions.Root.AddCommand(CommandLineOptions.Execute)

	//////////////////////////////////////////////////////////////////////////////
	// Describe
	if nil == CommandLineOptions.Describe.RunE {
		CommandLineOptions.Describe.RunE = func(cmd *cobra.Command, args []string) error {
			_, validateErr := govalidator.ValidateStruct(optionsDescribe)
			if nil != validateErr {
				return validateErr
			}

			fileWriter, fileWriterErr := os.Create(optionsDescribe.OutputFile)
			if fileWriterErr != nil {
				return fileWriterErr
			}
			defer fileWriter.Close()
			describeErr := Describe(serviceName,
				serviceDescription,
				lambdaAWSInfos,
				api,
				site,
				optionsDescribe.S3Bucket,
				OptionsGlobal.BuildTags,
				OptionsGlobal.LinkerFlags,
				fileWriter,
				workflowHooks,
				OptionsGlobal.Logger)

			if describeErr == nil {
				describeErr = fileWriter.Sync()
			}
			return describeErr
		}
	}
	CommandLineOptions.Root.AddCommand(CommandLineOptions.Describe)

	//////////////////////////////////////////////////////////////////////////////
	// Explore
	if nil == CommandLineOptions.Explore.RunE {
		CommandLineOptions.Explore.RunE = func(cmd *cobra.Command, args []string) error {
			_, validateErr := govalidator.ValidateStruct(optionsExplore)
			if nil != validateErr {
				return validateErr
			}
			return Explore(lambdaAWSInfos, optionsExplore.Port, OptionsGlobal.Logger)
		}
	}
	CommandLineOptions.Root.AddCommand(CommandLineOptions.Explore)

	//////////////////////////////////////////////////////////////////////////////
	// Profile
	if nil == CommandLineOptions.Profile.RunE {
		CommandLineOptions.Profile.RunE = func(cmd *cobra.Command, args []string) error {
			_, validateErr := govalidator.ValidateStruct(optionsProfile)
			if nil != validateErr {
				return validateErr
			}
			return Profile(serviceName,
				serviceDescription,
				optionsProfile.S3Bucket,
				optionsProfile.Port,
				OptionsGlobal.Logger)
		}
	}
	CommandLineOptions.Root.AddCommand(CommandLineOptions.Profile)
	// Run it!

	executeErr := CommandLineOptions.Root.Execute()
	if nil != OptionsGlobal.Logger && nil != executeErr {
		OptionsGlobal.Logger.Error(executeErr)
	}
	// Cleanup, if for some reason the caller wants to re-execute later...
	CommandLineOptions.Root.PersistentPreRunE = nil
	return executeErr
}
