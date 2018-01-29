var util = require('util')
var fs = require('fs')
var http = require('http')
var path = require('path')
var os = require('os')
var process = require('process')
var childProcess = require('child_process')
var spartaUtils = require('./sparta_utils')
var AWS = require('aws-sdk')
var awsConfig = new AWS.Config({})
var GOLANG_CONSTANTS = require('./golang-constants.json')
var proto = require('./proto/proxy_pb')

// TODO: See if https://forums.aws.amazon.com/message.jspa?messageID=633802
// has been updated with new information
process.env['PATH'] = process.env['PATH'] + ':' + process.env['LAMBDA_TASK_ROOT']

// Use the same binary name
var SPARTA_BINARY_NAME = 'Sparta.lambda.amd64'

// This name will be rewritten as part of the archive creation
var SPARTA_SERVICE_NAME = 'SpartaService'
// End dynamic reassignment

// This is where the binary will be extracted
var SPARTA_BINARY_PATH = util.format('./bin/%s', SPARTA_BINARY_NAME)
var SPARTA_LOG_LEVEL = 'info'

var MAXIMUM_RESPAWN_COUNT = 5

// Handle to the active golang process.
var golangProcess = null
var failCount = 0

var METRIC_NAMES = {
  CREATED: 'ProcessCreated',
  REUSED: 'ProcessReused',
  TERMINATED: 'ProcessTerminated'
}

var postRequestMetrics = function (path,
  startRemainingCountMillis,
  socketDuration,
  lambdaBodyLength,
  writeCompleteDuration,
  responseEndDuration) {
  var namespace = util.format('Sparta/%s', SPARTA_SERVICE_NAME)

  var params = {
    MetricData: [],
    Namespace: namespace
  }
  var dimensions = [
    {
      Name: 'Path',
      Value: path
    }
  ]
  // Log the uptime with every request...
  params.MetricData.push({
    MetricName: 'Uptime',
    Dimensions: dimensions,
    Unit: 'Seconds',
    Value: os.uptime()
  })
  params.MetricData.push({
    MetricName: 'StartRemainingTimeInMillis',
    Dimensions: dimensions,
    Unit: 'Milliseconds',
    Value: startRemainingCountMillis
  })
  params.MetricData.push({
    MetricName: 'LambdaResponseLength',
    Dimensions: dimensions,
    Unit: 'Bytes',
    Value: lambdaBodyLength
  })

  if (Array.isArray(socketDuration)) {
    params.MetricData.push({
      MetricName: util.format('OpenSocketDuration'),
      Dimensions: dimensions,
      Unit: 'Milliseconds',
      Value: Math.floor(socketDuration[0] / 1000 + socketDuration[1] * 1e9)
    })
  }

  if (Array.isArray(writeCompleteDuration)) {
    params.MetricData.push({
      MetricName: util.format('RequestCompleteDuration'),
      Dimensions: dimensions,
      Unit: 'Milliseconds',
      Value: Math.floor(writeCompleteDuration[0] / 1000 + writeCompleteDuration[1] * 1e9)
    })
  }

  if (Array.isArray(responseEndDuration)) {
    params.MetricData.push({
      MetricName: util.format('ResponseCompleteDuration'),
      Dimensions: dimensions,
      Unit: 'Milliseconds',
      Value: Math.floor(responseEndDuration[0] / 1000 + responseEndDuration[1] * 1e9)
    })
  }

  var cloudwatch = new AWS.CloudWatch(awsConfig)
  var onResult = function () {
    // NOP
  }
  cloudwatch.putMetricData(params, onResult)
}

function makeRequest (path, startRemainingCountMillis, event, context, lambdaCallback) {
  // http://docs.aws.amazon.com/lambda/latest/dg/nodejs-prog-model-context.html
  context.callbackWaitsForEmptyEventLoop = false

  // Let's track the request lifecycle
  var requestTime = process.hrtime()
  var lambdaBodyLength = 0
  var socketDuration = null
  var writeCompleteDuration = null
  var responseEndDuration = null

  // Let's go and turn the request into a proto
  var proxyRequest = new proto.AWSProxyRequest();

  // If there is a event.body element, try and parse it to make
  // interacting with API Gateway a bit simpler.  The .body property
  // corresponds to the data shape set by the *.vtl templates
  if (event && event.body) {
    try {
      event.body = JSON.parse(event.body)
    } catch (e) {}
  }
  // Set payload event
  stringifiedContent = JSON.stringify(event)
  proxyRequest.setEvent(Buffer.from(stringifiedContent).toString('base64'))
  // Set LambdaContext
  var lambdaContext = new proto.AWSLambdaContext()
  lambdaContext.setFunctionName(context.functionName)
  lambdaContext.setFunctionVersion(context.functionVersion)
  lambdaContext.setInvokedFunctionArn(context.invokedFunctionArn)
  lambdaContext.setMemoryLimitInMb(context.memoryLimitInMB)
  lambdaContext.setAwsRequestId(context.awsRequestId)
  lambdaContext.setLogGroupName(context.logGroupName)
  lambdaContext.setLogStreamName(context.logStreamName)
  if (context.identity) {
    cognitoIdentity = new proto.AWSCognitoIdentity()
    cognitoIdentity.setCognitoIdentityId(context.identity.cognitoIdentityId)
    cognitoIdentity.setCognitoIdentityPoolId(context.identity.cognitoIdentityPoolId)
    lambdaContext.setIdentity(cognitoIdentity)
  }
  if (context.clientContext) {
    clientContext = context.clientContext
    proxyClientContext = new proto.AWSClientContext()
    proxyClientContext.setInstallationId(clientContext.installation_id)
    proxyClientContext.setAppTitle(clientContext.app_title)
    proxyClientContext.setAppVersionName(clientContext.app_version_name)
    proxyClientContext.setAppVersionCode(clientContext.app_version_code)
    proxyClientContext.setAppPackageName(clientContext.app_package_name)
    if (clientContext.env) {
      clientContextEnv = clientContext.env
      proxyClientContextEnv = new proto.AWSClientContextEnv()
      proxyClientContextEnv.setPlatformVersion(clientContextEnv.platform_version)
      proxyClientContextEnv.setPlatform(clientContextEnv.platform)
      proxyClientContextEnv.setMake(clientContextEnv.make)
      proxyClientContextEnv.setModel(clientContextEnv.model)
      proxyClientContextEnv.setLocale(clientContextEnv.locale)
      proxyClientContext.setEnv(proxyClientContextEnv)
    }
  }
  proxyRequest.setContext(lambdaContext)

  // Setup request bytes
  var requestBytes = Buffer.from(proxyRequest.serializeBinary().buffer)
  var options = {
    host: 'localhost',
    port: 9999,
    path: path,
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-protobuf',
      'Content-Length': requestBytes.byteLength
    }
  }

  var onProxyComplete = function (err, response) {
    try {
      responseEndDuration = process.hrtime(requestTime)
      postRequestMetrics(path,
        startRemainingCountMillis,
        socketDuration,
        lambdaBodyLength,
        writeCompleteDuration,
        responseEndDuration)

      context.done(err, response)
    } catch (e) {
      context.done(e, null)
    }
  }

  var req = http.request(options, function (res) {
    res.setEncoding('utf8')
    var body = ''
    res.on('data', function (chunk) {
      body += chunk
    })
    res.on('end', function () {
      // Bridge the NodeJS and golang worlds by including the golang
      // HTTP status text in the error response if appropriate.  This enables
      // the API Gateway integration response to use standard golang StatusText regexp
      // matches to manage HTTP status codes.
      var responseData = {}
      var handlerError = (res.statusCode >= 400) ? new Error(body) : undefined
      if (handlerError) {
        responseData.code = res.statusCode
        responseData.status = GOLANG_CONSTANTS.HTTP_STATUS_TEXT[res.statusCode.toString()]
        responseData.headers = res.headers
        responseData.error = handlerError.toString()
      } else {
        responseData = body
        lambdaBodyLength = Buffer.byteLength(responseData, 'utf8')
        if (res.headers['content-type'] === 'application/json') {
          try {
            responseData = JSON.parse(body)
          } catch (e) {}
        }
      }
      var err = handlerError ? new Error(JSON.stringify(responseData)) : null
      var resp = handlerError ? null : responseData
      onProxyComplete(err, resp)
    })
  })
  req.once('socket', function (res) {
    socketDuration = process.hrtime(requestTime)
  })
  req.once('finish', function () {
    writeCompleteDuration = process.hrtime(requestTime)
  })
  req.once('error', function (e) {
    onProxyComplete(e, null)
  })
  req.write(requestBytes)
  req.end()
}

var postMetricCounter = function (metricName, userCallback) {
  var namespace = util.format('Sparta/%s', SPARTA_SERVICE_NAME)

  var params = {
    MetricData: [
      {
        MetricName: metricName,
        Unit: 'Count',
        Value: 1
      }
    ],
    Namespace: namespace
  }
  var cloudwatch = new AWS.CloudWatch(awsConfig)
  var onResult = function () {
    if (userCallback) {
      userCallback()
    }
  }
  cloudwatch.putMetricData(params, onResult)
}

var createForwarder = function (path) {
  var forwardToGolangProcess = function (event, context, callback, metricName, startRemainingCountMillisParam) {
    var startRemainingCountMillis = startRemainingCountMillisParam || context.getRemainingTimeInMillis()
    if (!golangProcess) {
      spartaUtils.log(util.format('Launching %s with args: execute --signal %d', SPARTA_BINARY_PATH, process.pid))
      golangProcess = childProcess.spawn(SPARTA_BINARY_PATH,
        ['execute',
          '--level',
          SPARTA_LOG_LEVEL,
          '--signal',
          process.pid], {
          'stdio': 'inherit'
        })
      var terminationHandler = function (eventName) {
        return function (value) {
          var onPosted = function () {
            console.error(util.format('Sparta %s: %s\n', eventName.toUpperCase(), JSON.stringify(value)))
            failCount += 1
            if (failCount > MAXIMUM_RESPAWN_COUNT) {
              process.exit(1)
            }
            golangProcess = null
            forwardToGolangProcess(null,
              null,
              callback,
              METRIC_NAMES.TERMINATED,
              startRemainingCountMillis)
          }
          postMetricCounter(METRIC_NAMES.TERMINATED, onPosted)
        }
      }
      golangProcess.on('error', terminationHandler('error'))
      golangProcess.on('exit', terminationHandler('exit'))
      process.on('exit', function () {
        spartaUtils.log('Go process exited')
        if (golangProcess) {
          golangProcess.kill()
        }
      })
      var golangProcessReadyHandler = function () {
        process.removeListener('SIGUSR2', golangProcessReadyHandler)
        forwardToGolangProcess(event,
          context,
          callback,
          METRIC_NAMES.CREATED,
          startRemainingCountMillis)
      }
      spartaUtils.log('Waiting for SIGUSR2 signal')
      process.on('SIGUSR2', golangProcessReadyHandler)
    }
    else if (event && context) {
      postMetricCounter(metricName || METRIC_NAMES.REUSED)
      makeRequest(path, startRemainingCountMillis, event, context, callback)
    }
  }
  return forwardToGolangProcess
}

// Log the outputs
var envSettings = {
  AWS_SDK_Version: AWS.VERSION,
  NodeJSVersion: process.version,
  Uptime: process.uptime(),
}
spartaUtils.log(envSettings)

exports.main = createForwarder('/')

// Additional golang handlers to be dynamically appended below
