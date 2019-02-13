package integration

import "time"

type StartTestRunPayload struct {
	Name       string                     `json:"name"`
	ProjectKey string                     `json:"projectKey"`
	Items      []TestCaseRunResultPayload `json:"items"`
}

type StartTestRunResponse struct {
	Key string `json:"key"`
}

type TestCaseRunResultPayload struct {
	TestCaseKey     string `json:"testCaseKey"`
	Status          string `json:"status"`
	ExecutionTime   int    `json:"executionTime"`
	ActualStartDate string `json:"actualStartDate"`
	ActualEndDate   string `json:"actualEndDate"`
}

type TestCaseRunResultResponse struct {
	Id int `json:"id"`
}

type TestSearchResponse struct {
	Name      string    `json:"name"`
	Key       string    `json:"key"`
	CreatedOn time.Time `json:"createdOn"`
}

type TestCaseGetResponse struct {
	Key        string `json:"key"`
	ProjectKey string `json:"projectKey"`
}

type TestEvent struct {
	Time    time.Time
	Action  string
	Package string
	Test    string
	Elapsed float64
	Output  string
}
