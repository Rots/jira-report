package cmd

import (
	"log"
	"path/filepath"
	"reports/burndown"
	"reports/jira"

	"github.com/spf13/cobra"
)

var (
	sprints                   []string
	board, output             string
	startMargin, fullTimeline bool
	workdayStart, workdayEnd  = 10, 18
)

func init() {
	burndownCmd.Flags().StringArrayVarP(&sprints, "sprint", "s", nil, "Name or ID of the sprint to get the report for. Default gets the first active sprint. If mutiple are provided, the output filenames are in the format {sprintID}-{output}.")
	burndownCmd.Flags().StringVarP(&board, "board", "b", "", "Name or ID of the Sprint board to use.")
	burndownCmd.Flags().StringVarP(&output, "output", "o", "estimates-burndown.html", "Name of the file to write the HTML report.")
	burndownCmd.Flags().BoolVar(&startMargin, "start-margin", startMargin, "add additional 1 day margin before the sprint start")
	burndownCmd.Flags().BoolVar(&fullTimeline, "full-timeline", fullTimeline, "Do not strip chart to working time only. Also show weekends and non-work time in the chart.")
	burndownCmd.Flags().IntVar(&workdayStart, "workday-start", workdayStart, "When does the working day start (0-23). This is ignored in full-timeline mode.")
	burndownCmd.Flags().IntVar(&workdayEnd, "workday-end", workdayEnd, "When does the working day end (0-23). This is ignored in full-timeline mode.")
	rootCmd.AddCommand(burndownCmd)
}

var burndownCmd = &cobra.Command{
	Use:     "burndown",
	Short:   "Generate the effort estimates burndown report for one or more given sprints.",
	Aliases: []string{"b"},
	Run: func(cmd *cobra.Command, args []string) {
		validateFlags()
		c := jira.InitJira(user, password, url)
		opts := burndown.Opts{
			Client:       c,
			Board:        board,
			Interactive:  interactive,
			StartMargin:  startMargin,
			FullTimeline: fullTimeline,
			WorkdayStart: workdayStart,
			WorkdayEnd:   workdayEnd,
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

func validateFlags() {
	if workdayStart < 0 || 23 < workdayStart {
		log.Fatalln("Workday start time must be in range 0-23")
	}
	if workdayEnd < 0 || 23 < workdayEnd {
		log.Fatalln("Workday end time must be in range 0-23")
	}
	if workdayEnd < workdayStart {
		log.Fatalln("Workday start time must be before end time")
	}
}

func fileNameWithPrefix(file, prefix string) string {
	dir := filepath.Dir(file)
	name := prefix + filepath.Base(file)
	return filepath.Join(dir, name)
}
