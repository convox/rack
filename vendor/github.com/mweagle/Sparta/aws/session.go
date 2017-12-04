package aws

import (
	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
)

type logrusProxy struct {
	logger *logrus.Logger
}

func (proxy *logrusProxy) Log(args ...interface{}) {
	proxy.logger.Info(args...)
}

// NewSessionWithConfig returns an awsSession that includes the user supplied
// configuration information
func NewSessionWithConfig(awsConfig *aws.Config, logger *logrus.Logger) *session.Session {
	return NewSessionWithConfigLevel(awsConfig, aws.LogDebugWithRequestErrors, logger)
}

// NewSession that attaches a debug level handler to all AWS requests from services
// sharing the session value.
func NewSession(logger *logrus.Logger) *session.Session {
	return NewSessionWithLevel(aws.LogDebugWithRequestErrors, logger)
}

// NewSessionWithLevel returns an AWS Session (https://github.com/aws/aws-sdk-go/wiki/Getting-Started-Configuration)
// object that attaches a debug level handler to all AWS requests from services
// sharing the session value.
func NewSessionWithLevel(level aws.LogLevelType, logger *logrus.Logger) *session.Session {
	awsConfig := &aws.Config{
		CredentialsChainVerboseErrors: aws.Bool(true),
	}
	return NewSessionWithConfigLevel(awsConfig, level, logger)
}

// NewSessionWithConfigLevel returns an AWS Session (https://github.com/aws/aws-sdk-go/wiki/Getting-Started-Configuration)
// object that attaches a debug level handler to all AWS requests from services
// sharing the session value.
func NewSessionWithConfigLevel(awsConfig *aws.Config,
	level aws.LogLevelType,
	logger *logrus.Logger) *session.Session {
	if nil == awsConfig {
		awsConfig = &aws.Config{
			CredentialsChainVerboseErrors: aws.Bool(true),
		}
	}

	// Log AWS calls if needed
	switch logger.Level {
	case logrus.DebugLevel:
		awsConfig.LogLevel = aws.LogLevel(level)
	}
	awsConfig.Logger = &logrusProxy{logger}
	sess := session.New(awsConfig)
	sess.Handlers.Send.PushFront(func(r *request.Request) {
		logger.WithFields(logrus.Fields{
			"Service":   r.ClientInfo.ServiceName,
			"Operation": r.Operation.Name,
			"Method":    r.Operation.HTTPMethod,
			"Path":      r.Operation.HTTPPath,
			"Payload":   r.Params,
		}).Debug("AWS Request")
	})

	logger.WithFields(logrus.Fields{
		"Name":    aws.SDKName,
		"Version": aws.SDKVersion,
	}).Debug("AWS SDK Info")

	return sess
}
