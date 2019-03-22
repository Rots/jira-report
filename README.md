# jira-report
Sprint burndown report based on remaining effort estimates.


## Installation

    git clone https://github.com/Rots/jira-report.git
    cd jira-report
    go build .

## Getting started

The tool provides help with regular `-h` flag.

To generate the report provide the JIRA URL and the target sprint ID.

Example:

    ./reports burndown --url https://jira.example.com --sprint 45
