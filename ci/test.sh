#!/bin/bash

function assert_run {
  run "$1" || { echo "failed"; exit 101; }
}

function fetch {
  fetch_once $1 && sleep 5 && fetch_once $1
}

function fetch_once {
  curl -ks --connect-timeout 5 --max-time 3 --retry 100 --retry-max-time 600 --retry-connrefused $1
}

function run {
  echo "running: $*" >&2
  eval $*
}

root="$(cd $(dirname ${0:-})/..; pwd)"

set -ex

provider=$(convox2 api get /system | jq -r .provider)

# cli
convox2 version

# rack
convox2 instances
convox2 rack
sleep 15
convox2 rack logs --no-follow | grep service/
convox2 rack ps | grep rack

# rack (provider-specific)
case $provider in
  aws)
    convox2 rack releases
    convox2 instances keyroll --wait
    instance=$(convox2 api get /instances | jq -r '.[0].id')
    convox2 instances ssh $instance "ls -la" | grep ec2-user
    convox2 instances terminate $instance
    convox2 rack | grep elb.amazonaws.com
    convox2 rack params | grep LogRetention
    convox2 rack params set LogRetention=14 --wait
    convox2 rack params | grep LogRetention | grep 14
    convox2 rack params set LogRetention= --wait
    convox2 rack params | grep LogRetention | grep -v 14
    ;;
esac

# app
cd $root/examples/httpd

if [ "${ACTION}" != "" ]; then
  # if update or downgrade test, it will deploy the app again
  convox2 deploy -a ci2 --wait
fi

convox2 apps | grep ci2
convox2 apps info ci2 | grep running
release=$(convox2 build -a ci2 -d cibuild --id) && [ -n "$release" ]
convox2 releases -a ci2 | grep $release
build=$(convox2 api get /apps/ci2/builds | jq -r ".[0].id") && [ -n "$build" ]
convox2 builds -a ci2 | grep $build
convox2 builds info $build -a ci2 | grep $build
convox2 builds info $build -a ci2 | grep cibuild
convox2 builds logs $build -a ci2 | grep "Running: docker push"
convox2 builds export $build -a ci2 -f /tmp/build.tgz
releasei=$(convox2 builds import -a ci2 -f /tmp/build.tgz --id) && [ -n "$releasei" ]
buildi=$(convox2 api get /apps/ci2/releases/$releasei | jq -r ".build") && [ -n "$buildi" ]
convox2 builds info $buildi -a ci2 | grep cibuild
echo "FOO=bar" | convox2 env set -a ci2
convox2 env -a ci2 | grep FOO | grep bar
convox2 env get FOO -a ci2 | grep bar
convox2 env unset FOO -a ci2
convox2 env -a ci2 | grep -v FOO
releasee=$(convox2 env set FOO=bar -a ci2 --id) && [ -n "$releasee" ]
convox2 env get FOO -a ci2 | grep bar
convox2 releases -a ci2 | grep $releasee
convox2 releases info $releasee -a ci2 | grep FOO
convox2 releases manifest $releasee -a ci2 | grep "build: ."
convox2 releases promote $release -a ci2 --wait
endpoint=$(convox2 api get /apps/ci2/services | jq -r '.[] | select(.name == "web") | .domain')
fetch https://$endpoint | grep "It works"
convox2 logs -a ci2 --no-follow | grep service/web
releaser=$(convox2 releases rollback $release -a ci2 --wait --id)
convox2 ps -a ci2 | grep $releaser
ps=$(convox2 api get /apps/ci2/processes | jq -r '.[]|select(.status=="running")|.id' | head -n 1)
convox2 ps info $ps -a ci2 | grep $releaser
convox2 scale web --count 2 --cpu 192 --memory 256 -a ci2 --wait
convox2 services -a ci2 | grep web | grep 443:80 | grep $endpoint
endpoint=$(convox2 api get /apps/ci2/services | jq -r '.[] | select(.name == "web") | .domain')
fetch https://$endpoint | grep "It works"
convox2 ps -a ci2 | grep web | wc -l | grep 2
ps=$(convox2 api get /apps/ci2/processes | jq -r '.[]|select(.status=="running")|.id' | head -n 1)
convox2 exec $ps "ls -la" -a ci2 | grep htdocs
cat /dev/null | convox2 exec $ps 'sh -c "sleep 2; echo test"' -a ci2 | grep test
convox2 run web "ls -la" -a ci2 | grep htdocs
cat /dev/null | convox2 run web 'sh -c "sleep 2; echo test"' -a ci2 | grep test
echo foo > /tmp/file
convox2 cp /tmp/file $ps:/file -a ci2
convox2 exec $ps "cat /file" -a ci2 | grep foo
mkdir -p /tmp/dir
echo foo > /tmp/dir/file
convox2 cp /tmp/dir $ps:/dir -a ci2
convox2 exec $ps "cat /dir/file" -a ci2 | grep foo
convox2 cp $ps:/dir /tmp/dir2 -a ci2
cat /tmp/dir2/file | grep foo
convox2 cp $ps:/file /tmp/file2 -a ci2
cat /tmp/file2 | grep foo
convox2 ps stop $ps -a ci2
convox2 ps -a ci2 | grep -v $ps
convox2 deploy -a ci2 --wait

# registries
convox2 registries
convox2 registries add quay.io convox+ci 6D5CJVRM5P3L24OG4AWOYGCDRJLPL0PFQAENZYJ1KGE040YDUGPYKOZYNWFTE5CV
convox2 registries | grep quay.io | grep convox+ci
convox2 build -a ci2 | grep -A 5 "Authenticating https://quay.io" | grep "Login Succeeded"
convox2 registries remove quay.io
convox2 registries | grep -v quay.io

# app (provider-specific)
case $provider in
  aws)
    convox2 apps params -a ci2 | grep LogRetention
    convox2 apps params set LogRetention=14 -a ci2 --wait
    convox2 apps params -a ci2 | grep LogRetention | grep 14
    convox2 apps params set LogRetention= -a ci2 --wait
    convox2 apps params -a ci2 | grep LogRetention | grep -v 14
    ;;
esac

# gen1
case $provider in
  aws)
    cd $root/examples/httpd
    convox2 apps create ci1 -g 1 --wait
    convox2 deploy -a ci1 --wait
    convox2 services -a ci1 | grep web | grep elb.amazonaws.com | grep 443:80
    endpoint=$(convox2 api get /apps/ci1/services | jq -r '.[] | select(.name == "web") | .domain')
    fetch https://$endpoint | grep "It works"
    ;;
esac

# test internal communication
case $provider in
  aws)
  convox2 apps delete ci1 --wait
  convox2 rack params set Internal=Yes

  cd $root/examples/httpd
  convox2 apps create ci1 --wait
  convox2 apps | grep ci1
  convox2 apps info ci1 | grep running
  convox2 deploy -a ci1 --wait
  convox2 apps info ci1 | grep running

  sleep 60

  rackname=$(convox2 rack | grep 'Name' | xargs | cut -d ' ' -f2 )

  sleep 10
  psci1=$(convox2 api get /apps/ci1/processes | jq -r '.[]|select(.status=="running" and .name == "web")|.id' | head -n 1)
  psci2=$(convox2 api get /apps/ci2/processes | jq -r '.[]|select(.status=="running" and .name == "web")|.id' | head -n 1)

  convox2 exec $psci1 "curl -k https://web.ci2.$rackname.convox" -a ci1 | grep "It works"
  convox2 exec $psci2 "curl -k https://web.ci1.$rackname.convox" -a ci2 | grep "It works"
    ;;
esac

# timers
sleep 30

timerLog=$(convox2 logs -a ci2 --no-follow --since 1m | grep service/example)
if ! [[ $timerLog == *"Hello Timer"* ]]; then
  echo "failed"; exit 1;
fi

# certs
# case $provider in
#   aws)
#     cd $root/ci/assets
#     convox2 certs
#     cert=$(convox2 certs generate example.org --id)
#     convox2 certs | grep -v $cert
#     convox2 certs delete $cert
#     cert=$(convox2 certs import example.org.crt example.org.key --id)
#     sleep 30
#     convox2 certs | grep $cert
#     certo=$(convox2 api get /apps/ci1/services | jq -r '.[] | select(.name == "web") | .ports[] | select (.balancer == 443) | .certificate')
#     convox2 ssl -a ci1 | grep web:443 | grep $certo
#     convox2 ssl update web:443 $cert -a ci1 --wait
#     convox2 ssl -a ci1 | grep web:443 | grep $cert
#     convox2 ssl update web:443 $certo -a ci1 --wait
#     convox2 ssl -a ci1 | grep web:443 | grep $certo
#     sleep 30
#     convox2 certs delete $cert
#     ;;
# esac
#
# # rack resources
# case $provider in
#   aws)
#     convox2 rack resources create syslog Url=tcp://syslog.convox.com --name cilog --wait
#     convox2 rack resources | grep cilog | grep syslog
#     convox2 rack resources info cilog | grep -v Apps
#     convox2 rack resources url cilog | grep tcp://syslog.convox.com
#     convox2 rack resources link cilog -a ci2 --wait
#     convox2 rack resources info cilog | grep Apps | grep ci2
#     convox2 rack resources unlink cilog -a ci2 --wait
#     convox2 rack resources info cilog | grep -v Apps
#     convox2 rack resources link cilog -a ci1 --wait
#     convox2 rack resources info cilog | grep Apps | grep ci1
#     convox2 rack resources unlink cilog -a ci1 --wait
#     convox2 rack resources info cilog | grep -v Apps
#     convox2 rack resources update cilog Url=tcp://syslog2.convox.com --wait
#     convox2 rack resources info cilog | grep syslog2.convox.com
#     convox2 rack resources url cilog | grep tcp://syslog2.convox.com
#     convox2 rack resources delete cilog --wait
#     convox2 rack resources create postgres --name pgdb --wait
#     convox2 rack resources | grep pgdb | grep postgres
#     dburl=$(convox2 rack resources url pgdb)
#     convox2 rack resources update pgdb BackupRetentionPeriod=2 --wait
#     [ "$dburl" == "$(convox2 rack resources url pgdb)" ]
#     convox2 rack resources delete pgdb --wait
#     ;;
# esac

# cleanup
convox2 apps delete ci2 --wait

# cleanup (provider-specific)
case $provider in
  aws)
    convox2 apps delete ci1 --wait
    ;;
esac
convox2 rack params set Internal=No
