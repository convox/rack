package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"/github.com/stretchr/testify/assert"
	"github.com/convox/kernel/controllers"
)

func TestNoPassword(t *testing.T) {
	aws := stubAws(DescribeConvoxStackCycle("convox-test"))
	defer aws.Close()
	defer os.Setenv("RACK", os.Getenv("RACK"))

	os.Setenv("RACK", "convox-test")

	assert.HTTPSuccess(t, controllers.HandlerFunc, "GET", "http://convox/system", nil)
}

func TestBasicAuth(t *testing.T) {
	assert := assert.New(t)
	aws := stubAws(DescribeConvoxStackCycle("convox-test"))
	defer aws.Close()
	defer os.Setenv("PASSWORD", os.Getenv("PASSWORD"))
	defer os.Setenv("RACK", os.Getenv("RACK"))

	os.Setenv("PASSWORD", "keymaster")
	os.Setenv("RACK", "convox-test")

	req, _ := http.NewRequest("GET", "http://convox/system", nil)
	w := httptest.NewRecorder()
	controllers.HandlerFunc(w, req)

	if !assert.Equal(401, w.Code) {
		return
	}

	w = httptest.NewRecorder()
	req.SetBasicAuth("", "keymaster")
	controllers.HandlerFunc(w, req)

	assert.Equal(200, w.Code)
}
