package sparta

import (
	"os"
	"testing"
)

func TestDescribe(t *testing.T) {
	logger, err := NewLogger("info")
	output, err := os.Create("./graph.html")
	if nil != err {
		t.Fatalf(err.Error())
		return
	}
	defer output.Close()
	err = Describe("SampleService", "SampleService Description", testLambdaData(), nil, nil, output, logger)
	if nil != err {
		t.Errorf("Failed to describe: %s", err)
	}
}
