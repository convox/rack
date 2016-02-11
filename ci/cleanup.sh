#!/bin/bash
set -x

export CIRCLE_ARTIFACTS=${CIRCLE_ARTIFACTS:-/tmp}
export CIRCLE_BUILD_NUM=${CIRCLE_BUILD_NUM:-0}

export STACK_NAME=c${CIRCLE_BUILD_NUM}
export TEMPLATE_FILE=api/dist/kernel.json

case $CIRCLE_NODE_INDEX in
  1)
  export AWS_DEFAULT_REGION=us-west-2
  export AWS_REGION=us-west-2
  ;;
  2)
  export AWS_DEFAULT_REGION=eu-west-1
  export AWS_REGION=eu-west-1
  ;;
  *)
  export AWS_DEFAULT_REGION=us-east-1
  export AWS_REGION=us-east-1
  ;;
esac


# Save Remaining ECS Artifacts
aws ecs list-clusters | tee $CIRCLE_ARTIFACTS/list-clusters.json
aws ecs describe-clusters --clusters $(jq -r '.clusterArns[]' $CIRCLE_ARTIFACTS/list-clusters.json) | tee $CIRCLE_ARTIFACTS/describe-clusters.json

for cluster in $(jq -r '.clusters[].clusterName' $CIRCLE_ARTIFACTS/describe-clusters.json); do
  aws ecs list-services     --cluster $cluster | tee $CIRCLE_ARTIFACTS/list-services-$cluster.json
  aws ecs describe-services --cluster $cluster --services $(jq -r '.serviceArns[]' $CIRCLE_ARTIFACTS/list-services-$cluster.json) | tee $CIRCLE_ARTIFACTS/describe-services-$cluster.json
done

# Save Remaining CloudWatch Logs Artifacts
aws logs describe-log-groups | tee $CIRCLE_ARTIFACTS/describe-log-groups.json

for groupName in $(jq -r ".logGroups[] | select(.logGroupName | contains(\"-$CIRCLE_BUILD_NUM-\")) | .logGroupName" $CIRCLE_ARTIFACTS/describe-log-groups.json); do
  aws logs describe-log-streams --log-group-name $groupName | tee $CIRCLE_ARTIFACTS/describe-log-streams-${groupName//\//-}.json

  for streamName in $(jq -r '.logStreams[].logStreamName' $CIRCLE_ARTIFACTS/describe-log-streams-${groupName//\//-}.json); do
    aws logs get-log-events --log-group-name $groupName --log-stream-name $streamName | jq '.events[]' | tee -a $CIRCLE_ARTIFACTS/get-log-events-${groupName//\//-}-${streamName//\//-}-unsorted.json
  done
done

# Save CF Artifacts
aws cloudformation list-stacks | tee $CIRCLE_ARTIFACTS/list-stacks.json

for stackId in $(jq -r ".StackSummaries[] | select(.StackId | contains(\"-$CIRCLE_BUILD_NUM\")) | .StackId" $CIRCLE_ARTIFACTS/list-stacks.json); do
  aws cloudformation describe-stacks       --stack-name $stackId | tee $CIRCLE_ARTIFACTS/describe-stacks-${stackId//\//-}.json
  aws cloudformation describe-stack-events --stack-name $stackId | tee $CIRCLE_ARTIFACTS/describe-stack-events-${stackId//\//-}.json
done

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
