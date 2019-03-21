package jira

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
	jira "gopkg.in/andygrunwald/go-jira.v1"
)

// Client is a local wrapper for remote JIRA client
type Client struct {
	*jira.Client
}

// InitJira creates the JIRA client instance authenticating with the provided credentials
// or requests password from `stdin` if not provided
func InitJira(user, pass, url string) *Client {
	if pass == "" {
		user, pass = getAuth(user)
	}
	auth := jira.BasicAuthTransport{Username: user, Password: pass}
	jiraClient, err := jira.NewClient(
		auth.Client(),
		url,
	)
	if err != nil {
		log.Fatal(err)
	}
	return &Client{jiraClient}
}

// getAuth splits user string into user and password based on first ':' or asks for password
func getAuth(user string) (string, string) {
	var pass string
	if strings.ContainsRune(user, ':') {
		tokens := strings.SplitN(user, ":", 2)
		user = tokens[0]
		pass = tokens[1]
	} else {
		if user == "" {
			fmt.Printf("Please provide user for JIRA: ")
			reader := bufio.NewReader(os.Stdin)
			var err error
			user, err = reader.ReadString('\n')
			if err != nil {
				log.Fatal(err)
			}
			user = strings.TrimSpace(user)
		}
		fmt.Printf("Please provide JIRA password for '%s': ", user)
		passBytes, err := terminal.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			log.Fatal(err)
		}

		log.Println("Password read.")
		pass = string(passBytes)
	}
	return user, pass
}
