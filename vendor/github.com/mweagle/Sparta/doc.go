// TODO: make `go generate` work on windows

//go:generate rm -rf ./resources/provision/node_modules
//go:generate npm install ./resources/provision/ --prefix ./resources/provision --silent
// Zip up the modules
//go:generate bash -c "pushd ./resources/provision; zip -qr ./node_modules.zip ./node_modules/"
//go:generate rm -rf ./resources/provision/node_modules

// Embed the custom service handlers
// TODO: Once AWS lambda supports golang as first class, move the
// NodeJS custom action helpers into golang
//go:generate go run $GOPATH/src/github.com/mjibson/esc/main.go -o ./CONSTANTS.go -private -pkg sparta ./resources
//go:generate go run ./resources/awsbinary/insertTags.go ./CONSTANTS !lambdabinary

// Create a secondary CONSTANTS_AWSBINARY.go file with empty content.  The next step will insert the
// build tags at the head of each file so that they are mutually exclusive, similar to the
// lambdabinaryshims.go file
//go:generate go run $GOPATH/src/github.com/mjibson/esc/main.go -o ./CONSTANTS_AWSBINARY.go -private -pkg sparta ./resources/awsbinary/README.md
//go:generate go run ./resources/awsbinary/insertTags.go ./CONSTANTS_AWSBINARY lambdabinary

// cleanup
//go:generate rm -f ./resources/provision/node_modules.zip

/*
Package sparta transforms a set of golang functions into an Amazon Lambda deployable unit.

The deployable archive includes

	 	1. NodeJS proxy logic
	 	2. A golang binary
	 	3. Dynamically generated CloudFormation template that supports create/update & delete operations.
	 	4. If specified, CloudFormation custom resources to automatically configure S3/SNS push registration
		5. If specified, API Gateway provisioning logic via custom resources to make the golang functions publicly accessible.

See the Main() docs for more information and examples
*/
package sparta
