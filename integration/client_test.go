package integration

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_StartTestRun(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.String(), "/testrun")

		var startPayload *StartTestRunPayload
		err := json.NewDecoder(r.Body).Decode(&startPayload)
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, "testrun1", startPayload.Name)
		assert.Equal(t, "testproject", startPayload.ProjectKey)
		assert.Equal(t,
			[]TestCaseRunResultPayload{
				{
					TestCaseKey:     "123",
					Status:          "fail",
					ExecutionTime:   1000,
					ActualStartDate: time.Now().Format(time.RFC3339),
					ActualEndDate:   time.Now().Format(time.RFC3339),
				},
			}, startPayload.Items,
		)

		re := &StartTestRunResponse{Key: "123"}
		data, _ := json.Marshal(re)
		_, _ = w.Write(data)
	}))
	defer ts.Close()
	c := New(ts.URL, "testproject", "testrun_example", "abc", "123", nil, "debug")
	ti := []TestCaseRunResultPayload{
		{
			TestCaseKey:     "123",
			Status:          "fail",
			ExecutionTime:   1000,
			ActualStartDate: time.Now().Format(time.RFC3339),
			ActualEndDate:   time.Now().Format(time.RFC3339),
		},
	}
	runKey, err := c.StartTestRun("testrun1", ti)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, runKey)
}

func TestClient_DeleteTestRun(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.String(), "/testrun/testrun_example")

		_, _ = w.Write([]byte("deleted"))
	}))
	c := New(ts.URL, "testproject", "testrun_example", "abc", "123", nil, "debug")
	err := c.DeleteTestRun("testrun_example")
	assert.NoError(t, err)
}

func TestClient_SearchTestRun(t *testing.T) {
	layout := "2006-01-02T15:04:05.000Z"
	str := "2014-11-12T11:45:26.371Z"
	someTime, err := time.Parse(layout, str)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.String(), "/testrun/search?fields=name%2Ckey%2CcreatedOn&maxResults=200&query=abc")

		re := &[]TestSearchResponse{{Name: "some_run", Key: "some_key", CreatedOn: someTime}}
		data, _ := json.Marshal(re)
		_, _ = w.Write(data)
	}))
	c := New(ts.URL, "testproject", "testrun_example", "abc", "123", nil, "debug")
	resp, err := c.SearchTestRun("abc")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []TestSearchResponse{{Name: "some_run", Key: "some_key", CreatedOn: someTime}}, resp)
}
