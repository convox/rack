package sparta

import (
	"fmt"
	"github.com/asaskevich/govalidator"
	"github.com/spf13/cobra"
	"os"
)

// NOTE: your application MUST use `package main` and define a `main()` function.  The
// example text is to make the documentation compatible with godoc.
// Should be main() in your application

// Additional command line options used for both the provision
// and CLI commands
type optionsStruct struct {
	Username   string `valid:"required,match(\\w+)"`
	Password   string `valid:"required,match(\\w+)"`
	SSHKeyName string `valid:"-"`
}

var options optionsStruct

// Common function to register shared command line flags
// across multiple Sparta commands
func registerSpartaCommandLineFlags(command *cobra.Command) {
	command.Flags().StringVarP(&options.Username,
		"username",
		"u",
		"",
		"HTTP Basic Auth username")
	command.Flags().StringVarP(&options.Password,
		"password",
		"p",
		"",
		"HTTP Basic Auth password")
}

func ExampleParseOptions() {
	//////////////////////////////////////////////////////////////////////////////
	// Add the custom command to run the sync loop
	syncCommand := &cobra.Command{
		Use:   "sync",
		Short: "Periodically perform a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Sync command!\n")
			return nil
		},
	}
	// Include the basic auth flags for the sync command
	registerSpartaCommandLineFlags(syncCommand)
	CommandLineOptions.Root.AddCommand(syncCommand)

	//////////////////////////////////////////////////////////////////////////////
	// Register custom flags for pre-existing Sparta commands
	registerSpartaCommandLineFlags(CommandLineOptions.Provision)
	CommandLineOptions.Provision.Flags().StringVarP(&options.SSHKeyName,
		"key",
		"k",
		"",
		"SSH Key Name to use for EC2 instances")

	//////////////////////////////////////////////////////////////////////////////
	// Define a validation hook s.t. we can validate the CLI user input
	validationHook := func(command *cobra.Command) error {
		if command.Name() == "provision" && len(options.SSHKeyName) <= 0 {
			return fmt.Errorf("SSHKeyName option is required")
		}
		fmt.Printf("Command: %s\n", command.Name())
		switch command.Name() {
		case "provision",
			"sync":
			_, validationErr := govalidator.ValidateStruct(options)
			return validationErr
		default:
			return nil
		}
	}
	// If the validation hooks failed, exit the application
	parseErr := ParseOptions(validationHook)
	if nil != parseErr {
		os.Exit(3)
	}
	//////////////////////////////////////////////////////////////////////////////
	//
	// Standard Sparta application
	// ...
}
