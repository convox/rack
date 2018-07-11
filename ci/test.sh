#!/bin/bash

function assert_run {
  run "$1" || { echo "failed"; exit 101; }
}

function fetch {
  hostname=$1
  c=0
  while ! curl -ks -m2 $hostname >/dev/null; do
    let c=c+1
    [ $c -gt 60 ] && exit 1
    sleep 20
  done
  sleep 10
  curl -ks https://$endpoint
}

function run {
  echo "running: $*" >&2
  eval $*
}

root="$(cd $(dirname ${0:-})/..; pwd)"

set -ex

# cli
convox version

# rack
convox instances
convox instances keyroll --wait
instance=$(convox api get /instances | jq -r '.[0].id')
convox instances ssh $instance "ls -la" | grep ec2-user
convox rack | grep elb.amazonaws.com
convox rack logs --no-follow | grep service/web
convox rack ps | grep bin/web
convox rack releases
convox rack params | grep LogRetention
convox rack params set LogRetention=14 --wait
convox rack params | grep LogRetention | grep 14
convox rack params set LogRetention= --wait
convox rack params | grep LogRetention | grep -v 14

# gen2
cd $root/examples/httpd
convox apps create ci2 --wait
convox apps | grep ci2
convox apps info ci2 | grep running
release=$(convox build -a ci2 -d cibuild --id) && [ -n "$release" ]
convox releases -a ci2 | grep $release
build=$(convox api get /apps/ci2/builds | jq -r ".[0].id") && [ -n "$build" ]
convox builds -a ci2 | grep $build
convox builds info $build -a ci2 | grep $build
convox builds info $build -a ci2 | grep cibuild
convox builds logs $build -a ci2 | grep "docker tag httpd"
convox builds export $build -a ci2 -f /tmp/build.tgz
releasei=$(convox builds import -a ci2 -f /tmp/build.tgz --id) && [ -n "$releasei" ]
buildi=$(convox api get /apps/ci2/releases/$releasei | jq -r ".build") && [ -n "$buildi" ]
convox builds info $buildi -a ci2 | grep cibuild
echo "FOO=bar" | convox env set -a ci2
convox env -a ci2 | grep FOO | grep bar
convox env get FOO -a ci2 | grep bar
convox env unset FOO -a ci2
convox env -a ci2 | grep -v FOO
releasee=$(convox env set FOO=bar -a ci2 --id) && [ -n "$releasee" ]
convox env get FOO -a ci2 | grep bar
convox releases -a ci2 | grep $releasee
convox releases info $releasee -a ci2 | grep FOO
convox releases manifest $releasee -a ci2 | grep "image: httpd"
convox releases promote $release -a ci2 --wait
endpoint=$(convox api get /apps/ci2/services | jq -r '.[] | select(.name == "web") | .domain')
fetch https://$endpoint | grep "It works"
convox logs -a ci2 --no-follow | grep service/web
releaser=$(convox releases rollback $release -a ci2 --wait --id)
convox ps -a ci2 | grep $releaser
ps=$(convox api get /apps/ci2/processes | jq -r '.[0].id')
convox ps info $ps -a ci2 | grep $releaser
convox scale web --count 2 --cpu 192 --memory 256 -a ci2 --wait
convox services -a ci2 | grep web | grep convox.site | grep 443:80
endpoint=$(convox api get /apps/ci2/services | jq -r '.[] | select(.name == "web") | .domain')
fetch https://$endpoint | grep "It works"
convox ps -a ci2 | grep web | wc -l | grep 2
convox run web "ls -la" -a ci2 | grep htdocs
ps=$(convox api get /apps/ci2/processes | jq -r '.[0].id')
convox exec $ps "ls -la" -a ci2 | grep htdocs
convox ps stop $ps
convox deploy -a ci2 --wait
convox apps params -a ci2 | grep LogRetention
convox apps params set LogRetention=14 -a ci2 --wait
convox apps params -a ci2 | grep LogRetention | grep 14
convox apps params set LogRetention= -a ci2 --wait
convox apps params -a ci2 | grep LogRetention | grep -v 14

# gen1
cd $root/examples/httpd
convox apps create ci1 -g 1 --wait
convox deploy -a ci1 --wait
convox services -a ci1 | grep web | grep elb.amazonaws.com | grep 443:80
endpoint=$(convox api get /apps/ci1/services | jq -r '.[] | select(.name == "web") | .domain')
fetch https://$endpoint | grep "It works"

# certs
cd $root/ci/assets
convox certs
cert=$(convox certs generate example.org --id)
convox certs | grep -v $cert
convox certs delete $cert
cert=$(convox certs import example.org.crt example.org.key --id)
sleep 30
convox certs | grep $cert
certo=$(convox api get /apps/ci1/services | jq -r '.[] | select(.name == "web") | .ports[] | select (.balancer == 443) | .certificate')
convox ssl -a ci1 | grep web:443 | grep $certo
convox ssl update web:443 $cert -a ci1 --wait
convox ssl -a ci1 | grep web:443 | grep $cert
convox ssl update web:443 $certo -a ci1 --wait
convox ssl -a ci1 | grep web:443 | grep $certo
sleep 30
convox certs delete $cert

# resources
convox resources create syslog Url=tcp://syslog.convox.com --name cilog --wait
convox resources | grep cilog | grep syslog
convox resources info cilog | grep -v Apps
convox resources url cilog | grep tcp://syslog.convox.com
convox resources link cilog -a ci2 --wait
convox resources info cilog | grep Apps | grep ci2
convox resources unlink cilog -a ci2 --wait
convox resources info cilog | grep -v Apps
convox resources link cilog -a ci1 --wait
convox resources info cilog | grep Apps | grep ci1
convox resources unlink cilog -a ci1 --wait
convox resources info cilog | grep -v Apps
convox resources update cilog Url=tcp://syslog2.convox.com --wait
convox resources info cilog | grep syslog2.convox.com
convox resources url cilog | grep tcp://syslog2.convox.com
convox resources delete cilog --wait

# cleanup
convox apps delete ci1 --wait
convox apps delete ci2 --wait
