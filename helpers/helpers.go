package helpers

import (
	"os"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/stvp/rollbar"
)

func init() {
	rollbar.Token = os.Getenv("ROLLBAR_TOKEN")
	// rollbar.Environment = "production" // defaults to "development"
}

func Error(log *logger.Logger, err error) {
	log.Error(err)
	rollbar.Error(rollbar.ERR, err)
}
