// +build !lambdabinary

package sparta

// Defines functions that are only valid in the context of the build
// binary
import (
	"fmt"
)

var codePipelineEnvironments map[string]map[string]string

func init() {
	codePipelineEnvironments = make(map[string]map[string]string)
}

// RegisterCodePipelineEnvironment is part of a CodePipeline deployment
// and defines the environments available for deployment. Environments
// are defined the `environmentName`. The values defined in the
// environmentVariables are made available to each service as
// environment variables. The environment key will be transformed into
// a configuration file for a CodePipeline CloudFormation action:
// TemplateConfiguration: !Sub "TemplateSource::${environmentName}".
func RegisterCodePipelineEnvironment(environmentName string, environmentVariables map[string]string) error {
	if _, exists := codePipelineEnvironments[environmentName]; exists {
		return fmt.Errorf("Environment (%s) has already been defined", environmentName)
	}
	codePipelineEnvironments[environmentName] = environmentVariables
	return nil
}
