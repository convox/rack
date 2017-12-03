var isString = function (value) {
  return typeof value === 'string'
}

module.exports.toBoolean = function (value) {
  var bValue = value
  if (isString(bValue)) {
    switch (bValue.toLowerCase().trim()) {
      case 'true':
      case '1':
        bValue = true
        break
      case 'false':
      case '0':
      case null:
        bValue = false
        break
      default:
        bValue = false
    }
  }
  return bValue
}

module.exports.idempotentDeleteHandler = function (successString, cb) {
  return function (e, results) {
    if (e) {
      if (e.toString().indexOf(successString) >= 0) {
        e = null
      }
    }
    cb(e, results || true)
  }
}
module.exports.cfnResponseLocalTesting = function () {
  console.log('Using local CFN response object')
  return {
    FAILED: 'FAILED',
    SUCCESS: 'SUCCESS',
    send: function (event, context, status, responseData) {
      var msg = {
        event: event,
        context: context,
        result: status,
        responseData: responseData
      }
      console.log(JSON.stringify(msg, null, ' '))
    }
  }
}

module.exports.log = function (objOrString) {
  if (isString(objOrString)) {
    try {
      // If it's empty, just skip it...
      if (objOrString.length <= 0) {
        return
      }
      objOrString = JSON.parse(objOrString)
    } catch (e) {
      // NOP
    }
  }
  if (isString(objOrString)) {
    objOrString = {msg: objOrString}
  }
  if (objOrString.stack) {
    console.error(objOrString)
  } else {
    console.log(JSON.stringify(objOrString))
  }
}
