#!/bin/bash
set -x

export CIRCLE_ARTIFACTS=${CIRCLE_ARTIFACTS:-/tmp/artifacts}
export CIRCLE_BUILD_NUM=${CIRCLE_BUILD_NUM:-0}
export STACK_NAME=convox-${CIRCLE_BUILD_NUM}
export $($(dirname $0)/region.sh ${CIRCLE_NODE_INDEX})

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
