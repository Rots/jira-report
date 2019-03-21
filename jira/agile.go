package jira

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	jira "gopkg.in/andygrunwald/go-jira.v1"
)

// GetScrumBoardID gets the jira ID of the given board name or if empty, interactively lets you choose
func (c *Client) GetScrumBoardID(board string, interact bool) (int, error) {
	r, err := strconv.Atoi(board)
	if err == nil {
		//If board is a number, we already found it
		return r, nil
	}
	//Else try to resolve by name
	v, err := c.getBoards()
	if err != nil {
		return 0, err
	}
	if board != "" {
		for _, b := range v.Values {
			if b.Name == board {
				return b.ID, nil
			}
		}
		return 0, errors.New("could not find a matching board for '" + board + "'")
	}
	if len(v.Values) == 0 {
		return 0, errors.New("no boards found")
	}
	if len(v.Values) == 1 {
		return v.Values[0].ID, nil
	}
	if !interact {
		var opts []string
		for _, b := range v.Values {
			opts = append(opts, fmt.Sprintf("%s (%v)", b.Name, b.ID))
		}
		return 0, errors.New("board must be selected, options are:" + strings.Join(opts, ","))
	}
	b, err := runInteractiveLoop(makeBoardOptions(v.Values), func(v interface{}) string {
		s := v.(jira.Board)
		return fmt.Sprintf("%s (%v)", s.Name, s.ID)
	})
	if err != nil {
		return 0, err
	}
	return b.(jira.Board).ID, nil
}

type options struct {
	items []interface{}
}

func makeBoardOptions(v []jira.Board) options {
	r := make([]interface{}, 0, len(v))
	for _, b := range v {
		r = append(r, b)
	}
	return options{r}
}
func makeSprintOptions(v []jira.Sprint) options {
	r := make([]interface{}, 0, len(v))
	for _, b := range v {
		r = append(r, b)
	}
	return options{r}
}

func runInteractiveLoop(options options, display func(interface{}) string) (interface{}, error) {
	switch len(options.items) {
	case 0:
		return nil, errors.New("no options provided")
	case 1:
		return options.items[0], nil
	}
	for {
		fmt.Println("Options:")
		for i, b := range options.items {
			fmt.Printf("%v: %s\n", i, display(b))
		}
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter choice: ")
		opt, _ := reader.ReadString('\n')
		s, err := strconv.Atoi(strings.TrimSpace(opt))
		if err == nil && s >= 0 && s < len(options.items) {
			return options.items[s], nil
		}
		fmt.Println(opt, err)
	}
}

// GetNumber converts string to number
func GetNumber(id string) (int, bool) {
	r, err := strconv.Atoi(id)
	return r, err == nil
}

// FindSprint retrieves the sprint data based on the given name or ID of the sprint
func (c *Client) FindSprint(board int, sprintName string, interactive bool) (jira.Sprint, error) {
	id, ok := GetNumber(sprintName)
	if ok {
		return c.GetSprint(id)
	}
	// Else try match the name on the selected board
	sprints, _, err := c.Board.GetAllSprints(fmt.Sprint(board))
	if err != nil {
		return jira.Sprint{}, err
	}
	for _, s := range sprints {
		if s.Name == sprintName {
			return s, nil
		}
	}

	if interactive {
		result, err := runInteractiveLoop(makeSprintOptions(sprints), func(v interface{}) string {
			s := v.(jira.Sprint)
			var activeSuf string
			if s.State == "active" {
				activeSuf = " (Active)"
			}
			return fmt.Sprintf("%s (%v)", s.Name, s.ID) + activeSuf
		})
		if err != nil {
			return jira.Sprint{}, err
		}
		return result.(jira.Sprint), nil
	}
	return jira.Sprint{}, fmt.Errorf("no sprint '%s' found", sprintName)
}

// GetSprint gets the sprint with given ID
func (c *Client) GetSprint(sprintID int) (jira.Sprint, error) {
	apiEndpoint := fmt.Sprintf("rest/agile/1.0/sprint/%d", sprintID)

	req, err := c.NewRequest("GET", apiEndpoint, nil)

	if err != nil {
		return jira.Sprint{}, err
	}

	var result jira.Sprint
	resp, err := c.Do(req, &result)
	if err != nil {
		err = jira.NewJiraError(resp, err)
	}

	return result, err
}

func (c *Client) getBoards() (*jira.BoardsList, error) {
	bs, _, err := c.Board.GetAllBoards(nil)
	return bs, err
}

// GetActiveSprint gets an active sprint
func (c *Client) GetActiveSprint(board int) (jira.Sprint, error) {
	s, _, err := c.Board.GetAllSprintsWithOptions(board, &jira.GetAllSprintsOptions{State: "active"})
	if err != nil {
		return jira.Sprint{}, err
	}
	//assume we have at least one active sprint, use the first entry
	if len(s.Values) == 0 {
		return jira.Sprint{}, errors.New("there are no active sprints")
	}
	return s.Values[0], err
}
