// +build !lambdabinary,!noop

package cgo

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"

	"strings"

	"path"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/mweagle/Sparta"
	spartaAWS "github.com/mweagle/Sparta/aws"
	"github.com/spf13/cobra"
	"golang.org/x/tools/go/ast/astutil"
)

// cgoMain is the internal handler for the cgo.Main*() functions. It's responsible
// for rewriting the incoming source file s.t. it can be compiled
// as a CGO library
func cgoMain(callerMainInputFilepath string,
	serviceName string,
	serviceDescription string,
	lambdaAWSInfos []*sparta.LambdaAWSInfo,
	api *sparta.API,
	site *sparta.S3Site,
	workflowHooks *sparta.WorkflowHooks) error {

	// We need to parse the command line args and get the subcommand...
	cgoCommandName := ""
	validationHook := func(command *cobra.Command) error {
		cgoCommandName = command.Name()
		return nil
	}
	// Extract & validate the SSH Key
	parseErr := sparta.ParseOptions(validationHook)
	if nil != parseErr {
		return fmt.Errorf("Failed to parse command line")
	}

	// We can short circuit a lot of this if we're just
	// trying to do a unit test or export.
	if cgoCommandName != "provision" {
		return sparta.MainEx(serviceName,
			serviceDescription,
			lambdaAWSInfos,
			api,
			site,
			workflowHooks,
			true)
	}

	// This is the provision workflow, which to make CGO
	// compatible in a C-lib context depends on being able
	// to rewrite the main() function so that the main() contents
	// happen in the context of an init() statement. That init()
	// statement will handle initializing the HTTP dispatch
	// map that's accessed as part of the Python caller
	// Read the main() input
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, callerMainInputFilepath, nil, 0)
	if err != nil {
		fmt.Printf("ParseFileErr: %s", err.Error())
		return err
	}

	// Add the imports that we need in the walkers text
	astutil.AddImport(fset, file, "unsafe")

	// default cgo package alias
	cgoPackageAlias := "cgo"
	// Did the user supply a cgo alias?
	for _, eachImport := range astutil.Imports(fset, file) {
		for _, eachImportEntry := range eachImport {
			if "\"github.com/mweagle/Sparta/cgo\"" == eachImportEntry.Path.Value && nil != eachImportEntry.Name {
				cgoPackageAlias = eachImportEntry.Name.String()
				break
			}
		}
	}
	// Great, now change main to init()
	for _, eachVisitor := range visitors() {
		ast.Walk(eachVisitor, file)
	}

	// Now save the file as a string and run it through
	// the text transformers
	var byteWriter bytes.Buffer
	transformErr := printer.Fprint(&byteWriter, fset, file)
	if nil != transformErr {
		return transformErr
	}
	updatedSource := byteWriter.String()
	for _, eachTransformer := range transformers() {
		transformedSource, transformedSourceErr := eachTransformer(updatedSource, cgoPackageAlias)
		if nil != transformedSourceErr {
			return transformedSourceErr
		}
		updatedSource = transformedSource
	}
	// The temporary file is the input file, with a suffix
	rewrittenFilepath := fmt.Sprintf("%s.sparta-cgo.go", callerMainInputFilepath)
	originalInputRenamedFilepath := fmt.Sprintf("%s.sparta.og", callerMainInputFilepath)
	renameErr := os.Rename(callerMainInputFilepath, originalInputRenamedFilepath)
	if nil != renameErr {
		fmt.Printf("Failed to backup source: %s", renameErr.Error())
		return renameErr
	}
	defer os.Rename(originalInputRenamedFilepath, callerMainInputFilepath)

	outputFile, outputFileErr := os.Create(rewrittenFilepath)
	if nil != outputFileErr {
		fmt.Printf("Failed to create output file: %s", outputFileErr.Error())
		return outputFileErr
	}

	// Save the updated contents
	_, writtenErr := outputFile.WriteString(updatedSource)
	if nil != writtenErr {
		return writtenErr
	}
	outputFile.Close()

	// Great, let's go ahead and do the build.
	spartaErr := sparta.MainEx(serviceName,
		serviceDescription,
		lambdaAWSInfos,
		api,
		site,
		workflowHooks,
		true)

	if nil == spartaErr {
		// Move it to the scratch location s.t. users can see what
		// was generated
		workingDir, err := os.Getwd()
		if nil != err {
			os.Remove(rewrittenFilepath)
		} else {
			originalInputFilename := path.Base(callerMainInputFilepath)
			scratchPath := filepath.Join(workingDir, sparta.ScratchDirectory, originalInputFilename)
			os.Rename(rewrittenFilepath, scratchPath)
		}

	} else {
		preservedOutput := strings.TrimSuffix(rewrittenFilepath, ".go")
		os.Rename(rewrittenFilepath, preservedOutput)
	}
	return spartaErr
}

// NewSession returns an AWS Session when running locally
func NewSession() *session.Session {
	logger, _ := sparta.NewLogger("info")
	return spartaAWS.NewSession(logger)
}
