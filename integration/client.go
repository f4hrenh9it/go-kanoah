package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

type Client struct {
	BaseURL     *url.URL
	Project     string
	TestRunName string
	JiraUser    string
	JiraPasswd  string
	Events      []*TestEvent

	httpClient *http.Client
	l          *zap.SugaredLogger
}

func New(baseUrl string, project string, testrunName string, jiraUser string, jiraPasswd string, client *http.Client, verbosity string) *Client {
	if client == nil {
		client = &http.Client{}
	}
	u, err := url.Parse(baseUrl)
	if err != nil {
		log.Fatal(err)
	}
	return &Client{
		BaseURL:     u,
		Project:     project,
		TestRunName: testrunName,
		JiraUser:    jiraUser,
		JiraPasswd:  jiraPasswd,
		httpClient:  client,
		l:           NewLogger(verbosity),
	}
}

func (c *Client) StartTestRun(name string, testItems []TestCaseRunResultPayload) (string, error) {
	p := StartTestRunPayload{
		Name:       name,
		ProjectKey: c.Project,
		Items:      testItems,
	}
	req, err := c.newRequest("POST", "testrun", p, nil)
	if err != nil {
		return "", err
	}
	c.l.Debug("starting test run", "project", c.Project, "name", p.Name)
	var trr StartTestRunResponse
	_, err = c.do(req, &trr)
	if err != nil {
		return "", err
	}
	c.l.Infof("test run started", "key", trr.Key)
	return trr.Key, nil
}

func (c *Client) DeleteTestRun(key string) error {
	req, err := c.newRequest("DELETE", fmt.Sprintf("testrun/%s", key), nil, nil)
	if err != nil {
		return err
	}
	_, err = c.do(req, nil)
	if err != nil {
		return err
	}
	c.l.Info("test run deleted")
	return nil
}

func (c *Client) SearchTestRun(query string) ([]TestSearchResponse, error) {
	req, err := c.newRequest("GET", "testrun/search", nil, map[string]string{"query": query, "maxResults": "200", "fields": "name,key,createdOn"})
	if err != nil {
		return nil, err
	}
	var tsr []TestSearchResponse
	_, err = c.do(req, &tsr)
	if err != nil {
		return nil, err
	}
	c.l.Debugf("search results: %s", tsr)
	return tsr, nil
}

func (c *Client) CheckTestCaseExists(testCaseId string) error {
	u := fmt.Sprintf("testcase/%s", testCaseId)
	req, err := c.newRequest("GET", u, nil, nil)
	if err != nil {
		return err
	}
	var tcr TestCaseGetResponse
	_, err = c.do(req, &tcr)
	if err != nil {
		return err
	}
	c.l.Debugf("get test case results: %s", tcr)
	return nil
}

func (c *Client) newRequest(method, path string, body interface{}, queryParams map[string]string) (*http.Request, error) {
	rel := &url.URL{Path: path}
	u := c.BaseURL.ResolveReference(rel)
	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.BasicAuth())
	var q url.Values
	if queryParams != nil {
		q = req.URL.Query()
		for k, v := range queryParams {
			q.Add(k, v)
		}
	}
	req.URL.RawQuery = q.Encode()
	return req, nil
}

func (c *Client) do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		bb, _ := ioutil.ReadAll(resp.Body)
		c.l.Errorf("request failed: status: %s, body: %s", resp.Status, string(bb))
		return nil, responseErr
	}
	if v != nil {
		err = json.NewDecoder(resp.Body).Decode(v)
	}
	return resp, err
}
