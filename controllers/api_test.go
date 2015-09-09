package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/convox/kernel/controllers"
)

func TestNoAuth(t *testing.T) {
	req, _ := http.NewRequest("GET", "", nil)
	w := httptest.NewRecorder()
	controllers.SingleRequest(w, req)

	if w.Code != 301 {
		t.Errorf("expected status code of %d, got %d", 301, w.Code)
		return
	}
}
