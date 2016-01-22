# CI

Scripts that install convox, deploy and app, delete and app, uninstall convox, and log artifacts.

These scripts are run on CircleCI (see circle.yml), though intended to aid local testing too.

## AWS / CircleCI Setup

Create an isolated AWS account. Copy keys into CircleCI environment variables:

- AWS_ACCESS_KEY_ID
- AWS_SECRET_ACCESS_KEY

Also set a region into CircleCI environment variables:

- AWS_DEFAULT_REGION
- AWS_REGION

Add GitHub convox/rack to CircleCI. 

We do not want to run AWS tests on all pushes, so disconnect convox/rack from CircleCI on the GitHub side.

See the [release ci script](https://github.com/convox/release/blob/master/bin/ci) to see how these tests are invoked for a branch.

## Manual Cleaning

* Log into CI AWS Management Console

Delete CloudFormation Stacks

* Manually delete all stacks (CREATE_COMPLETE, DELETE_FAILED, ROLLBACK_CLEANUP_FAILED, etc.) in CloudFormation
* In some cases this will not proceed because of DELETE_FAILED on certain resources. Manually deleting that resource through the Management Console should help get through this.
* In extreme cases an AWS Support Ticket may be required to delete the stack

Delete Leaked AWS Resources

* S3 Buckets
* CloudWatch LogGroups for Lambda Functions
* ECR Repositories
* ECS Task Definitions

