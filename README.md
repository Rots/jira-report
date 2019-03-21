# jira-report
Sprint burndown report based on remaining effort estimates.


## Installation

    go get -u github.com/Rots/jira-report

## Getting started

The tool provides help with regular `-h` flag.

To generate the report provide the JIRA URL and the target sprint ID.

Example:

    ./reports burndown --url https://jira.example.com --sprint 45
