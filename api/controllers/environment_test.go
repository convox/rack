package controllers_test

// got close but failed timebox on figuring out cycle tests

// func TestEnvironmentSetWhileCreating(t *testing.T) {
//   aws := stubAws(test.DescribeAppStackCycle("foo"))
//   defer aws.Close()

//   w := httptest.NewRecorder()

//   r, err := buildRequest("POST", "http://convox/apps/foo/environment", url.Values{})
//   require.Nil(t, err)

//   r.Header.Add("Version", "dev")

//   controllers.HandlerFunc(w, r)

//   assert.Equal(t, 500, w.Code)
//   assert.Equal(t, "", w.Body.String())
// }
