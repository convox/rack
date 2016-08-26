package models_test

import (
	"bytes"

	"github.com/convox/logger"
)

func init() {
	var buf bytes.Buffer
	logger.Output = &buf
}
