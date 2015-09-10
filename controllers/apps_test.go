package controllers_test

/*
func TestAppList(t *testing.T) {
	s := httptest.NewServer(awsutil.NewHandler([]awsutil.Cycle{
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  ".ListTasks",
				Body:       `{"cluster":"convox"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"taskArns":["arn:aws:ecs:us-east-1:901416387788:task/320a8b6a-c243-47d3-a1d1-6db5dfcb3f58"]}`,
			},
		},
	}))
	defer s.Close()

	aws.DefaultConfig.Region = "test"
	aws.DefaultConfig.Endpoint = s.URL

	os.Setenv("CLUSTER", "convox")
	os.Setenv("DYNAMO_RELEASES", "releases")
	os.Setenv("TEST_DOCKER_HOST", s.URL)

	req, _ := http.NewRequest("GET", "http://convox/apps", nil)
	w := httptest.NewRecorder()
	controllers.SingleRequest(w, req)

	t.Logf("%d - %s", w.Code, w.Body.String())

	if w.Code != 200 {
		t.Errorf("expected status code of %d, got %d", 200, w.Code)
		return
	}
}

func TestAppCreate(t *testing.T) {
	req, _ := http.NewRequest("GET", "", nil)
	w := httptest.NewRecorder()
	err := controllers.AppCreate(w, req)

	t.Logf("%d - %s", w.Code, w.Body.String())

	if err != nil {
		t.Error(err.Error())
		return
	}

	if w.Code != 200 {
		t.Errorf("expected status code of %d", 200)
		return
	}
}
*/
