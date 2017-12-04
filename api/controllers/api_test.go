package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/provider"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

// Note: these tests don't use the api helpers to ensure a naked
//       client can connect

func TestNoPassword(t *testing.T) {
	Mock(func(p *provider.MockProvider) {
		p.On("SystemGet").Return(nil, nil)

		aws := test.StubAws(test.DescribeConvoxStackCycle("convox-test"))
		defer aws.Close()
		defer os.Setenv("RACK", os.Getenv("RACK"))

		os.Setenv("RACK", "convox-test")

		assert.HTTPSuccess(t, controllers.HandlerFunc, "GET", "http://convox/system", nil)
	})
}

func TestBasicAuth(t *testing.T) {
	Mock(func(p *provider.MockProvider) {
		p.On("SystemGet").Return(nil, nil)

		assert := assert.New(t)
		aws := test.StubAws(test.DescribeConvoxStackCycle("convox-test"))
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
	})
}
