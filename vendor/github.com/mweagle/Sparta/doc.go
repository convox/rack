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
