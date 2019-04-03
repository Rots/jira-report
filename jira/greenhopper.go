package jira

import (
	"fmt"
	"strings"
	"time"

	jira "gopkg.in/andygrunwald/go-jira.v1"
)

type BoardInfo struct {
	ID             int
	Name           string
	WeekDays       map[time.Weekday]bool
	NonWorkingDays []time.Time
}

type boardInfo struct {
	ID                int               `json:"id"`
	Name              string            `json:"name"`
	WorkingDaysConfig workingDaysConfig `json:"workingDaysConfig"`
}
type workingDaysConfig struct {
	WeekDays       map[string]interface{} `json:"weekDays"`
	NonWorkingDays []isoDate              `json:"nonWorkingDays"` //>iso8601Date"`
}
type isoDate struct {
	Date date `json:"iso8601Date"`
}

type date time.Time

func (d *date) UnmarshalJSON(b []byte) error {
	time, err := time.Parse("2006-01-02", strings.Trim(string(b), "\" "))
	if err != nil {
		return err
	}
	*d = date(time)
	return nil
}

// GetBoardInfo retrieves the data about the board
func (c *Client) GetBoardInfo(id int) (BoardInfo, error) {
	apiEndpoint := "rest/greenhopper/1.0/rapidviewconfig/editmodel.json?rapidViewId=%v"
	req, err := c.NewRequest("GET", fmt.Sprintf(apiEndpoint, id), nil)
	if err != nil {
		return BoardInfo{}, err
	}

	var result boardInfo
	resp, err := c.Do(req, &result)
	if err != nil {
		return BoardInfo{}, jira.NewJiraError(resp, err)
	}

	return convertToPublic(result), nil

}

var days = map[string]time.Weekday{
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
	"sunday":    time.Sunday,
}

func convertToPublic(b boardInfo) BoardInfo {
	weekdays := make(map[time.Weekday]bool, 7)

	for k, v := range b.WorkingDaysConfig.WeekDays {
		isWeekday, ok := v.(bool)
		if ok {
			weekdays[days[k]] = isWeekday
		}
	}

	nonWork := make([]time.Time, 0, len(b.WorkingDaysConfig.NonWorkingDays))
	for _, v := range b.WorkingDaysConfig.NonWorkingDays {
		nonWork = append(nonWork, time.Time(v.Date))
	}

	return BoardInfo{
		ID:             b.ID,
		Name:           b.Name,
		WeekDays:       weekdays,
		NonWorkingDays: nonWork,
	}
}
