from __future__ import print_function
import pprint
import json
import sys
import traceback
import os
import base64
import boto3
from ctypes import *
from botocore.credentials import get_credentials
from botocore.session import get_session

import logging
logger = logging.getLogger()
logger.setLevel(logging.INFO)
initLogDict = {
    "Python Version": sys.version,
    "AWS SDK Version": boto3.__version__
  }
logging.info(json.dumps(initLogDict))
SPARTA_LOG_LEVEL = "{{ .LogLevel }}"

libHandle = None
try:
    libHandle = cdll.LoadLibrary("./bin/{{ .LibraryName }}")
    libHandle.Lambda.argtypes = [c_char_p,
                            c_char_p,
                            c_char_p,
                            c_char_p,
                            c_char_p,
                            c_char_p,
                            POINTER(c_int),
                            c_char_p,
                            c_int,
                            c_char_p,
                            c_int]
    libHandle.Lambda.restype = c_int
except:
    traceback.print_exc()
    message = "Unexpected error: " + sys.exc_info()[0]
    print(message)
    raise RuntimeError(message)

################################################################################
# AWS Lambda limits
# Ref: http://docs.aws.amazon.com/lambda/latest/dg/limits.html
################################################################################
MAX_RESPONSE_SIZE = 6 * 1024 * 1024
response_buffer = create_string_buffer(MAX_RESPONSE_SIZE)
MAX_RESPONSE_CONTENT_TYPE_SIZE = 1024
response_content_type_buffer = create_string_buffer(MAX_RESPONSE_CONTENT_TYPE_SIZE)


def lambda_handler(funcName, event, context):
    try:
        # Need to marshall the string into something we can get to in the
        # Go universe, so for that we can just get a struct
        # with the context. The event content can be passed in as a
        # raw char pointer.

        # Base64 encode the event data because that's what
        # proto expects for bytes data
        eventJSON = json.dumps(event).encode('utf-8')
        base64string = base64.b64encode(eventJSON).decode('utf-8')
        request = dict(event=base64string)

        contextDict = dict(
            functionName=context.function_name,
            functionVersion=context.function_version,
            invokedFunctionArn=context.invoked_function_arn,
            memoryLimitInMb=context.memory_limit_in_mb,
            awsRequestId=context.aws_request_id,
            logGroupName=context.log_group_name,
            logStreamName=context.log_stream_name
        )

        # Identity check...
        identityContext = getattr(context, "identity", None)
        if identityContext is not None:
            identityDict = dict()
            if getattr(identityDict, "cognito_identity_id", None):
                identityDict["cognitoIdentityId"] = identityContext["cognito_identity_id"]
            if getattr(identityDict, "cognito_identity_pool_id", None):
                identityDict["cognitoIdentityPoolId"] = identityContext["cognito_identity_pool_id"]

            contextDict["identity"] = identityDict

        # Client context
        if getattr(context, "client_context", None):
            awsClientContext = context.client_context
            contextDict["client_context"] = dict(
                installation_id=awsClientContext.installation_id,
                app_title=awsClientContext.app_title,
                app_version_name=awsClientContext.app_version_name,
                app_version_code=awsClientContext.app_version_code,
                Custom=awsClientContext.custom,
                env=awsClientContext.env
            )

        # Update it
        request["context"] = contextDict
        memset(response_buffer, 0, MAX_RESPONSE_SIZE)
        memset(response_content_type_buffer, 0, MAX_RESPONSE_CONTENT_TYPE_SIZE)
        exitCode = c_int()

        credentials = get_credentials(get_session())

        logger.debug('Sending event: {}'.format(request))
        bytesWritten = libHandle.Lambda(funcName.encode('utf-8'),
                                    SPARTA_LOG_LEVEL.encode('utf-8'),
                                    json.dumps(request).encode('utf-8'),
                                    credentials.access_key.encode('utf-8'),
                                    credentials.secret_key.encode('utf-8'),
                                    credentials.token.encode('utf-8'),
                                    byref(exitCode),
                                    response_content_type_buffer,
                                    MAX_RESPONSE_CONTENT_TYPE_SIZE-1,
                                    response_buffer,
                                    MAX_RESPONSE_SIZE-1)

        logger.debug('Lambda exit code: {}'.format(exitCode))
        if exitCode.value != 0:
            raise Exception(response_content_type_buffer.value)

        lowercase_content_type = response_content_type_buffer.value.lower()
        if "json" in lowercase_content_type.decode('utf-8'):
            try:
                json_object = json.loads(response_buffer.value)
                return json_object
            except:
                # They claim it's JSON, but it's not. Be nice
                return response_buffer.value.decode('utf-8')
        else:
            return response_buffer.value.decode('utf-8')
    except:
        traceback.print_exc()
        print("ERROR:", sys.exc_info()[0])
        raise

## Insert auto generated code here...
{{ .PythonFunctions }}
