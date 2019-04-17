package cmd

import (
	"path/filepath"
	"reports/burndown"
	"reports/jira"

	"github.com/spf13/cobra"
)

var (
	sprints                   []string
	board, output             string
	startMargin, fullTimeline bool
)

func init() {
	burndownCmd.Flags().StringArrayVarP(&sprints, "sprint", "s", nil, "Name or ID of the sprint to get the report for. Default gets the first active sprint. If mutiple are provided, the output filenames are in the format {sprintID}-{output}.")
	burndownCmd.Flags().StringVarP(&board, "board", "b", "", "Name or ID of the Sprint board to use.")
	burndownCmd.Flags().StringVarP(&output, "output", "o", "estimates-burndown.html", "Name of the file to write the HTML report.")
	burndownCmd.Flags().BoolVar(&startMargin, "start-margin", startMargin, "add additional 1 day margin before the sprint start")
	burndownCmd.Flags().BoolVar(&fullTimeline, "full-timeline", fullTimeline, "Do not strip chart to working time only. Also show weekends and non-work time in the chart.")
	rootCmd.AddCommand(burndownCmd)
}

var burndownCmd = &cobra.Command{
	Use:     "burndown",
	Short:   "Generate the effort estimates burndown report for one or more given sprints.",
	Aliases: []string{"b"},
	Run: func(cmd *cobra.Command, args []string) {
		c := jira.InitJira(user, password, url)
		opts := burndown.Opts{
			Client:       c,
			Board:        board,
			Interactive:  interactive,
			StartMargin:  startMargin,
			FullTimeline: fullTimeline,
		}
		switch len(sprints) {
		case 0:
			opts.Outfile = output
			burndown.Run(opts)
			return
		case 1:
			opts.Outfile = output
			opts.Sprint = sprints[0]
			burndown.Run(opts)
			return
		}
		for _, s := range sprints {
			opts.Outfile = fileNameWithPrefix(output, s+"-")
			opts.Sprint = s
			burndown.Run(opts)
		}
	},
}

func fileNameWithPrefix(file, prefix string) string {
	dir := filepath.Dir(file)
	name := prefix + filepath.Base(file)
	return filepath.Join(dir, name)
}
