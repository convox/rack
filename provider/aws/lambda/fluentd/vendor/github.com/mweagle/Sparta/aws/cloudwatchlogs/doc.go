// Package cloudwatchlogs scopes CloudWatchLogs-specific utiltities for
// Sparta
package cloudwatchlogs

/*
Log message format, data is
Base64 encoded and compressed with the gzip format
{ awslogs: { data: '...' } }

{
  "messageType": "DATA_MESSAGE",
  "owner": "123412341234",
  "logGroup": "/aws/lambda/versions",
  "logStream": "2016/02/20/[$LATEST]5efb6fc38f214f89827218367e12b37b",
  "subscriptionFilters": ["SpartaApplication_MyFilterc1a4c95a445ebf504a2d1a99f25e6ff3d459ea9f"],
  "logEvents": [{
    "id": "32468509038720158048534573168074737027928689874631065600",
    "timestamp": 1455938299356,
    "message": "START RequestId: 96f98a63-d780-11e5-ab78-69015eb2dceb Version: $LATEST\n"
  }, {
    "id": "32468509038787060284130165037499344182746634959149006849",
    "timestamp": 1455938299359,
    "message": "2016-02-20T03:18:19.358Z\t96f98a63-d780-11e5-ab78-69015eb2dceb\tNodeJS v.v0.10.36, AWS SDK v.2.2.32\n"
  }, {
    "id": "32468509038787060284130165037499344182746634959149006850",
    "timestamp": 1455938299359,
    "message": "END RequestId: 96f98a63-d780-11e5-ab78-69015eb2dceb\n"
  }, {
    "id": "32468509038787060284130165037499344182746634959149006851",
    "timestamp": 1455938299359,
    "message": "REPORT RequestId: 96f98a63-d780-11e5-ab78-69015eb2dceb\tDuration: 0.52 ms\tBilled Duration: 100 ms \tMemory Size: 128 MB\tMax Memory Used: 13 MB\t\n"
  }]
}
*/
