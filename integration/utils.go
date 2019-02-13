package integration

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

func (c *Client) GetLatestRunKey(testRuns []TestSearchResponse) string {
	sort.Slice(testRuns, func(i, j int) bool {
		return testRuns[i].CreatedOn.After(testRuns[j].CreatedOn)
	})
	if len(testRuns) == 0 {
		return ""
	}
	c.l.Debugf("latest run key: %s", testRuns[0].Key)
	return testRuns[0].Key
}

func (c *Client) FilterByName(name string, testRuns []TestSearchResponse) []TestSearchResponse {
	filtered := []TestSearchResponse{}
	for _, v := range testRuns {
		if v.Name == name {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func (c *Client) UpdateLatestTestRun(testCasesResults []TestCaseRunResultPayload) error {
	testruns, err := c.SearchTestRun(fmt.Sprintf("projectKey = \"%s\"", c.Project))
	if err != nil {
		return err
	}
	filteredRuns := c.FilterByName(c.TestRunName, testruns)
	latestKey := c.GetLatestRunKey(filteredRuns)
	if latestKey != "" {
		err = c.DeleteTestRun(latestKey)
		if err != nil {
			return err
		}
	}

	_, err = c.StartTestRun(c.TestRunName, testCasesResults)
	if err != nil {
		return err
	}
	return nil
}

func (m *Client) ParseEvents(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		te := &TestEvent{}
		err := json.Unmarshal(scanner.Bytes(), te)
		if err != nil {
			log.Fatal(err)
		}
		m.Events = append(m.Events, te)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	m.l.Debugf("total events parsed: %d", len(m.Events))
}

func (c *Client) GroupEventsByPackage() map[string][]*TestEvent {
	groupedEvents := make([]*TestEvent, 0)
	eventsByPackage := make(map[string][]*TestEvent)
	testNames := make(map[string]int)
	for _, e := range c.Events {
		if _, ok := testNames[e.Test]; !ok {
			testNames[e.Test] = 1
		}
	}
	for uniqTest := range testNames {
		for _, event := range c.Events {
			if event.Test == uniqTest && event.Test != "" {
				groupedEvents = append(groupedEvents, event)
			}
		}
	}
	packages := make(map[string]int)
	for _, e := range groupedEvents {
		if _, ok := packages[e.Package]; !ok {
			packages[e.Package] = 1
		}
	}
	c.l.Debugf("total packages: %d", len(packages))
	if len(packages) == 0 {
		c.l.Fatalf("no packages with tests found, skipping report")
	}
	for uniqPackage := range packages {
		for _, event := range groupedEvents {
			if event.Package == uniqPackage && event.Package != "" {
				eventsByPackage[uniqPackage] = append(eventsByPackage[uniqPackage], event)
			}
		}
	}
	return eventsByPackage
}

func (m *Client) DeleteBrokenTests(eventsByPackages map[string][]*TestEvent) map[string]map[string][]*TestEvent {
	tests := map[string]map[string][]*TestEvent{}
	for _, p := range eventsByPackages {
		for _, e := range p {
			if tests[e.Package] == nil {
				tests[e.Package] = make(map[string][]*TestEvent)
			}
			tests[e.Package][e.Test] = append(tests[e.Package][e.Test], e)
		}
	}
	for packName, p := range tests {
		m.l.Debugf("package", "package", packName)
		for testName, t := range p {
			m.l.Debugf("test", "test", testName)
			for _, e := range t {
				m.l.Debugf("event", "test", e.Test, "action", e.Action, "package", e.Package)
			}
		}
	}
	goodTests := []string{}
	for _, p := range tests {
		for _, t := range p {
			testWithEnd := false
			testWithStart := false
			for _, e := range t {
				if e.Action == "pass" || e.Action == "fail" || e.Action == "skip" {
					testWithEnd = true
				}
				if e.Action == "run" {
					testWithStart = true
				}
			}
			if !testWithEnd {
				m.l.Errorf("endless test", "test", t[0].Test)
			}
			if !testWithStart {
				m.l.Errorf("startless test", "test", t[0].Test)
			}
			if testWithStart && testWithEnd {
				goodTests = append(goodTests, t[0].Test)
			}
		}
	}
	finalTests := map[string]map[string][]*TestEvent{}
	for _, p := range tests {
		for testName, t := range p {
			for _, e := range t {
				if finalTests[e.Package] == nil {
					finalTests[e.Package] = make(map[string][]*TestEvent)
				}
				if stringInSlice(testName, goodTests) {
					finalTests[e.Package][e.Test] = append(finalTests[e.Package][e.Test], e)
				}
			}
		}
	}
	return finalTests
}

func (c *Client) Tests2TestResultsPayloads(tests map[string]map[string][]*TestEvent) []TestCaseRunResultPayload {
	results := []TestCaseRunResultPayload{}
	caseRe, _ := regexp.Compile("testcase_id:(.+-T.+)")
	for _, p := range tests {
		for testName, t := range p {
			startTime, endTime := c.getTimeBounds(t)
			var caseId [][]string
			for _, e := range t {
				if caseId == nil {
					caseId = caseRe.FindAllStringSubmatch(e.Output, -1)
				}
				if len(caseId) > 0 && len(caseId[0]) > 1 && (e.Action == "fail" || e.Action == "pass" || e.Action == "skip") {
					if err := c.CaseExistsInKanoah(caseId[0][1]); err != nil {
						c.l.Errorf("testcase id is not found in kanoah: %s, test item will be skipped", caseId[0][1])
						continue
					}
					results = append(results, TestCaseRunResultPayload{
						TestCaseKey:     caseId[0][1],
						Status:          strings.Title(e.Action),
						ExecutionTime:   int(e.Elapsed),
						ActualStartDate: startTime.Format(time.RFC3339),
						ActualEndDate:   endTime.Format(time.RFC3339),
					})
				}
			}
			if caseId == nil {
				c.l.Errorf("testcase id is not found in test %s, use fmt.Println(`case_id:COR-T63`) in test to provide case id reference", testName)
			}
		}
	}
	c.l.Debugf("matching with kanoah cases found: %s", results)
	return results
}

func (m *Client) getTimeBounds(events []*TestEvent) (time.Time, time.Time) {
	b := make([]*TestEvent, len(events))
	copy(b, events)
	sort.Slice(b, func(i, j int) bool {
		return b[i].Time.Before(b[j].Time)
	})
	return b[0].Time, b[len(b)-1].Time
}

func (c *Client) CaseExistsInKanoah(caseId string) error {
	if err := c.CheckTestCaseExists(caseId); err != nil {
		return err
	}
	return nil
}

func (c *Client) BasicAuth() string {
	b64 := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.JiraUser, c.JiraPasswd)))
	return fmt.Sprintf("Basic %s", b64)
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
