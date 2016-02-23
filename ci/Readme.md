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
* CloudWatch LogGroups (from Lambda Functions)
* ECR Repositories and Images
* ECS Task Definitions (make inactive)
* KMS Keys

## Cleanup Script

If you have the `aws` cli configured with CI creds and the `jq` utility, this script may work.

```bash
#!/bin/bash

set -x
export AWS_DEFAULT_PROFILE=ci

aws s3api list-buckets | jq '.Buckets[] | select(.Name | startswith("convox") or startswith("httpd") or startswith("node-workers")) | .Name' | xargs -n1 -I{} aws s3 rb --force s3://{}

aws logs describe-log-groups | jq .logGroups[].logGroupName | xargs -n1 aws logs delete-log-group --log-group-name

aws ecr describe-repositories | jq .repositories[].repositoryName | xargs -n1 aws ecr delete-repository --force --repository-name

aws ecs list-task-definitions --status active | jq .taskDefinitionArns[] | xargs -n1 aws ecs deregister-task-definition --task-definition

for key in $(aws kms list-keys | jq -r .Keys[].KeyId); do
  aws kms describe-key --key-id $key | jq -e '.KeyMetadata.KeyState == "Disabled"' && \
    aws kms schedule-key-deletion --pending-window-in-days 7 --key-id $key
done
```