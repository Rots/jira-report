package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "reports",
	Short: "JIRA report generator",
}

var (
	user, url   string
	password    string
	interactive bool
)

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&user, "user", "u", "", "The user to log into JIRA as. Use 'user:pass' to skip interactive password prompt.")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "The password to use for JIRA user. Also see '--user'.")
	rootCmd.PersistentFlags().StringVar(&url, "url", url, "JIRA URL")
	rootCmd.PersistentFlags().BoolVarP(&interactive, "interactive", "i", interactive, "Enable interactive prompts")
	rootCmd.MarkPersistentFlagRequired("url")
}
