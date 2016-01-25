#!/bin/bash
set -x

export CIRCLE_ARTIFACTS=${CIRCLE_ARTIFACTS:-/tmp}
export CIRCLE_BUILD_NUM=${CIRCLE_BUILD_NUM:-0}

export STACK_NAME=convox-${CIRCLE_BUILD_NUM}
export TEMPLATE_FILE=api/dist/kernel.json

# Clean leaked S3 Buckets, Repositories and Log Groups
aws s3api list-buckets |\
  jq ".Buckets[] | select(.Name | contains(\"-$CIRCLE_BUILD_NUM-\")) | .Name" |\
  xargs -L1 -I% aws s3 rb --force s3://%

aws ecr describe-repositories |\
  jq ".repositories[] | select(.repositoryName | contains(\"-$CIRCLE_BUILD_NUM-\")) | .repositoryName" |\
  xargs -L1 aws ecr delete-repository --force --repository-name

aws logs describe-log-groups |\
  jq ".logGroups[] | select(.logGroupName | contains(\"-$CIRCLE_BUILD_NUM-\")) | .logGroupName" |\
  xargs -L1 aws logs delete-log-group --log-group-name

# Save ECS Artifacts
aws ecs list-clusters | tee $CIRCLE_ARTIFACTS/list-clusters.json
aws ecs describe-clusters --clusters $(jq -r '.clusterArns[]' $CIRCLE_ARTIFACTS/list-clusters.json) | tee $CIRCLE_ARTIFACTS/describe-clusters.json

for cluster in $(jq -r '.clusters[].clusterName' $CIRCLE_ARTIFACTS/describe-clusters.json); do
  aws ecs list-services     --cluster $cluster | tee $CIRCLE_ARTIFACTS/list-services-$cluster.json
  aws ecs describe-services --cluster $cluster --services $(jq -r '.serviceArns[]' $CIRCLE_ARTIFACTS/list-services-$cluster.json) | tee $CIRCLE_ARTIFACTS/describe-services-$cluster.json
done

# Save Lambda Artifacts
aws logs describe-log-groups --log-group-name-prefix /aws/lambda/convox-$CIRCLE_BUILD_NUM | tee $CIRCLE_ARTIFACTS/describe-log-groups.json
groupName=$(jq -r '.logGroups[].logGroupName' $CIRCLE_ARTIFACTS/describe-log-groups.json)
aws logs describe-log-streams --log-group-name $groupName | tee $CIRCLE_ARTIFACTS/describe-log-streams.json

for streamName in $(jq -r '.logStreams[].logStreamName' $CIRCLE_ARTIFACTS/describe-log-streams.json); do
  aws logs get-log-events --log-group-name $groupName --log-stream-name $streamName | jq '.events[]' | tee -a $CIRCLE_ARTIFACTS/get-log-events-unsorted.json
done

jq -s 'sort_by(.timestamp)' $CIRCLE_ARTIFACTS/get-log-events-unsorted.json > $CIRCLE_ARTIFACTS/get-log-events.json

# Save CF Artifacts

# describe possible orphan kernel and app stacks from latest build
for stack in $(aws cloudformation list-stacks | jq -r ".StackSummaries[] | select(.StackName | endswith(\"-$CIRCLE_BUILD_NUM\")) | .StackName"); do
  aws cloudformation describe-stacks       --stack-name $stack | tee $CIRCLE_ARTIFACTS/describe-stacks-$stack.json
  aws cloudformation describe-stack-events --stack-name $stack | tee $CIRCLE_ARTIFACTS/describe-stack-events-$stack.json
done

# describe possible DELETE_COMPLETE kernel and app stacks from latest build
aws cloudformation list-stacks --stack-status-filter DELETE_COMPLETE | jq -r ".StackSummaries[] | select(.StackName | endswith(\"-$CIRCLE_BUILD_NUM\"))" | tee $CIRCLE_ARTIFACTS/list-stacks-delete-complete.json

for stack in $(jq -r '.StackName' $CIRCLE_ARTIFACTS/list-stacks-delete-complete.json); do
  stackId=$(jq -r "select (.StackName==\"$stack\") | .StackId" $CIRCLE_ARTIFACTS/list-stacks-delete-complete.json)
  aws cloudformation describe-stacks       --stack-name $stackId | tee $CIRCLE_ARTIFACTS/describe-stacks-$stack.json
  aws cloudformation describe-stack-events --stack-name $stackId | tee $CIRCLE_ARTIFACTS/describe-stack-events-$stack.json
done
