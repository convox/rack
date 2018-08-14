#!/bin/bash
set -ex -o pipefail

export CIRCLE_ARTIFACTS=${CIRCLE_ARTIFACTS:-/tmp/artifacts}
export CIRCLE_BUILD_NUM=${CIRCLE_BUILD_NUM:-0}
export $($(dirname $0)/region.sh ${CIRCLE_NODE_INDEX})

# hack to delete lambda enis
vpc=$(aws ec2 describe-vpcs --filter Name=tag:Name,Values=convox-${CIRCLE_BUILD_NUM} --query "Vpcs[0].VpcId" --output text)
enis=$(aws ec2 describe-network-interfaces  --filters Name=vpc-id,Values=$vpc --query "NetworkInterfaces[?contains(@.Description, 'Lambda')].NetworkInterfaceId" --output text)

if [ "$enis" != "" ]; then
  echo $enis | while read eni; do
    attachment=$(aws ec2 describe-network-interfaces --filters Name=network-interface-id,Values=$eni --query "NetworkInterfaces[0].Attachment.AttachmentId" --output text)

    if [ "$attachment" != "None" ]; then
      aws ec2 detach-network-interface --attachment-id $attachment
      sleep 10
    fi

    aws ec2 delete-network-interface --network-interface-id $eni
  done
fi

convox rack uninstall aws convox-${CIRCLE_BUILD_NUM} --force
