package main

import (
	"flag"
	"fmt"
	"github.com/f4hrenh9it/go-kanoah/integration"
	"log"
)

var version string

func main() {
	reportFilename := flag.String("json_report", "", "go test json report file path")
	jiraApiUrl := flag.String("jira_api_url", "https://jit.ozon.ru/rest/atm/1.0/", "jira api url")
	project := flag.String("kanoah_project", "", "kanoah project name")
	testrunName := flag.String("kanoah_testrun", "", "kanoah project name")
	jiraUser := flag.String("jira_user", "", "jira user")
	jiraPasswd := flag.String("jira_passwd", "", "jira passwd")
	logLevel := flag.String("log_level", "", "logging level")
	flag.Parse()
	if *project == "" {
		log.Fatal("provide your kanoah project name")
	}
	if *jiraUser == "" || *jiraPasswd == "" {
		log.Fatal("provide your jira user and password, ex. --jira_user abc --jira_passwd 123")
	}
	fmt.Printf("ver: %s\n", version)
	c := integration.New(*jiraApiUrl, *project, *testrunName, *jiraUser, *jiraPasswd, nil, *logLevel)
	c.ParseEvents(*reportFilename)
	tests := c.GroupEventsByPackage()
	filteredTests := c.DeleteBrokenTests(tests)
	ti := c.Tests2TestResultsPayloads(filteredTests)
	if err := c.UpdateLatestTestRun(ti); err != nil {
		log.Fatal(err)
	}
}
