#!/bin/bash

set -ex -o pipefail

declare -a RESOURCES=("s3" "sns" "sqs" "mysql")

# syslog resource
convox rack resources create syslog Url=tcp://syslog.convox.com --name cilog --wait
convox rack resources | grep cilog | grep syslog
convox rack resources info cilog | grep -v Apps
convox rack resources url cilog | grep tcp://syslog.convox.com
convox rack resources link cilog -a ci2 --wait
convox rack resources info cilog | grep Apps | grep ci2
convox rack resources unlink cilog -a ci2 --wait
convox rack resources info cilog | grep -v Apps
convox rack resources link cilog -a ci1 --wait
convox rack resources info cilog | grep Apps | grep ci1
convox rack resources unlink cilog -a ci1 --wait
convox rack resources info cilog | grep -v Apps
convox rack resources update cilog Url=tcp://syslog2.convox.com --wait
convox rack resources info cilog | grep syslog2.convox.com
convox rack resources url cilog | grep tcp://syslog2.convox.com
convox rack resources delete cilog --wait

# postgres resource
convox rack resources create postgres --name pgdb --wait
convox rack resources | grep pgdb | grep postgres
dburl=$(convox rack resources url pgdb)
convox rack resources update pgdb BackupRetentionPeriod=2 --wait
[ "$dburl" == "$(convox rack resources url pgdb)" ]
convox rack resources delete pgdb --wait

# create all 4 resources
for i in "${RESOURCES[@]}"
do
  convox rack resources create $i
done

for i in "${RESOURCES[@]}"
do
  # Check for resource to be marked as running
  j=0
  while [ "$(convox rack resources | grep $i | grep running | wc -l)" != "1" ]
  do
    # Exit if it takes more than 15 minutes
    # mysql can take up to 15 minutes to create
    if [ $((j++)) -gt 75 ]; then
      exit 1
    fi
    echo "Waiting for resource $i to be marked as running..."
    sleep 10
  done
done

# delete all 4 resources
for i in "${RESOURCES[@]}"
do
  name=$(convox rack resources | grep $i | awk '{print $1}')
  echo "deleting resource $name"
  convox rack resources delete $name --wait
done
