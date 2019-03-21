package jira

import (
	jira "gopkg.in/andygrunwald/go-jira.v1"
)

// GetAllStatuses return all available states in JIRA
func (c *Client) GetAllStatuses() ([]jira.Status, error) {
	apiEndpoint := "rest/api/2/status"
	req, err := c.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, err
	}

	var result []jira.Status
	resp, err := c.Do(req, &result)
	if err != nil {
		return nil, jira.NewJiraError(resp, err)
	}

	return result, err
}
