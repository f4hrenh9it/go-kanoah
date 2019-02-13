##### Go test json report to Kanoah integration

```
go get -u github.com/f4hrenh9it/go-kanoah
go-kanoah -json_report integration/testdata/parallel-report.json -jira_user abc -jira_passwd 123 -kanoah_project COR -kanoah_testrun testrun-branch-1 -log_level error
```