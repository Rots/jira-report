package cmd

import (
	"path/filepath"
	"reports/burndown"
	"reports/jira"

	"github.com/spf13/cobra"
)

var (
	sprints       []string
	board, output string
	startMargin   bool
)

func init() {
	burndownCmd.Flags().StringArrayVarP(&sprints, "sprint", "s", nil, "Name or ID of the sprint to get the report for. Default gets the first active sprint. If mutiple are provided, the output filenames are in the format {sprintID}-{output}.")
	burndownCmd.Flags().StringVarP(&board, "board", "b", "", "Name or ID of the Sprint board to use.")
	burndownCmd.Flags().StringVarP(&output, "output", "o", "estimates-burndown.html", "Name of the file to write the HTML report.")
	burndownCmd.Flags().BoolVar(&startMargin, "start-margin", startMargin, "add additional 24h margin before the sprint start")
	rootCmd.AddCommand(burndownCmd)
}

var burndownCmd = &cobra.Command{
	Use:     "burndown",
	Short:   "Generate the effort estimates burndown report for one or more given sprints.",
	Aliases: []string{"b"},
	Run: func(cmd *cobra.Command, args []string) {
		c := jira.InitJira(user, password, url)
		switch len(sprints) {
		case 0:
			burndown.Run(c, board, "", interactive, output, startMargin)
			return
		case 1:
			burndown.Run(c, board, sprints[0], interactive, output, startMargin)
			return
		}
		for _, s := range sprints {
			burndown.Run(c, board, s, interactive, fileNameWithPrefix(output, s+"-"), startMargin)
		}
	},
}

func fileNameWithPrefix(file, prefix string) string {
	dir := filepath.Dir(file)
	name := prefix + filepath.Base(file)
	return filepath.Join(dir, name)
}
