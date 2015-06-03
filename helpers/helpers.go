package helpers

import (
	"os"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/stvp/rollbar"
)

func init() {
	rollbar.Token = os.Getenv("ROLLBAR_TOKEN")
}

func Error(log *logger.Logger, err error) {
	if log == nil {
		log.Error(err)
	}
	rollbar.Error(rollbar.ERR, err)
}
