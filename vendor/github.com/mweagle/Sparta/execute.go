package sparta

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"net/http"
	"os"
	"path"
	"time"
)

// Port used for HTTP proxying communication
const defaultHTTPPort = 9999

// Execute creates an HTTP listener to dispatch execution. Typically
// called via Main() via command line arguments.
func Execute(lambdaAWSInfos []*LambdaAWSInfo, port int, parentProcessPID int, logger *logrus.Logger) error {
	validationErr := validateSpartaPreconditions(lambdaAWSInfos, logger)
	if validationErr != nil {
		return validationErr
	}

	if port <= 0 {
		port = defaultHTTPPort
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      NewLambdaHTTPHandler(lambdaAWSInfos, logger),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	logger.WithFields(logrus.Fields{
		"ParentPID": parentProcessPID,
	}).Info("Signaling parent process")

	if 0 != parentProcessPID {
		platformKill(parentProcessPID)
	}
	binaryName := path.Base(os.Args[0])
	logger.WithFields(logrus.Fields{
		"URL": fmt.Sprintf("http://localhost:%d", port),
	}).Info(fmt.Sprintf("Starting %s server", binaryName))

	err := server.ListenAndServe()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Error("Failed to launch server")
		return err
	}

	return nil
}
