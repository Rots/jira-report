package burndown

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	agile "reports/jira"
	"sort"
	"strconv"
	"strings"
	"time"

	jira "gopkg.in/andygrunwald/go-jira.v1"
)

type data struct {
	start                              time.Time
	progressCategory, completeCategory map[string]bool
	inProgress                         []entry
	new                                []entry
}
type entry struct {
	Time  time.Time
	Value int
	Msg   string
}

type diagram struct {
	Sprint    jira.Sprint
	Entries   []tableEntry
	StartLine bool
}
type tableEntry struct {
	Time     time.Time
	New      int
	Progress int
}

// Run creates the burndown report for remaining effort for given sprint
// sprint can be provided as JIRA internal sprint ID or as sprint name
func Run(j *agile.Client, board, sprint string, interactive bool, outfile string, startMargin bool) {
	s, err := getSprint(j, board, sprint, interactive)
	if err != nil {
		log.Fatalln(err)
	}
	var start time.Time
	if s.StartDate != nil {
		start = *s.StartDate
	}
	data := &data{start: start}
	if startMargin {
		data.start = data.start.Add(-24 * time.Hour)
	}
	data.progressCategory, data.completeCategory, err = getStates(j)
	if err != nil {
		log.Fatalln(err)
	}
	opts := &jira.SearchOptions{Expand: "changelog"}
	err = j.Issue.SearchPages(fmt.Sprintf("Sprint = %v ", s.ID), opts, data.collect)
	if err != nil {
		log.Fatalln(err)
	}

	f, err := os.Create(outfile)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	sum := data.collapse(data.start)
	printDiagram(f, diagram{s, sum, startMargin})
	printTable(f, "New", data.new)
	printTable(f, "Progress", data.inProgress)
	log.Println("Report written to: " + outfile)
}

func getStates(j *agile.Client) (map[string]bool, map[string]bool, error) {
	cat, _, err := j.StatusCategory.GetList()
	if err != nil {
		return nil, nil, err
	}
	var progID, doneID int
	for _, c := range cat {
		if c.ColorName == "yellow" {
			progID = c.ID
		}
		if c.ColorName == "green" {
			doneID = c.ID
		}
	}
	//Categorise all statuses
	states, err := j.GetAllStatuses()
	if err != nil {
		return nil, nil, err
	}
	progress, done := make(map[string]bool), make(map[string]bool)
	for _, s := range states {
		if s.StatusCategory.ID == progID {
			progress[s.Name] = true
		}
		if s.StatusCategory.ID == doneID {
			done[s.Name] = true
		}
	}
	return progress, done, nil
}

func getSprint(j *agile.Client, board, sprint string, interactive bool) (jira.Sprint, error) {
	sprintID, isID := agile.GetNumber(sprint)
	if !isID {
		b, err := j.GetScrumBoardID(board, interactive)
		if err != nil {
			return jira.Sprint{}, err
		}
		if sprint == "" && !interactive {
			return j.GetActiveSprint(b)
		}
		return j.FindSprint(b, sprint, interactive)
	}
	return j.GetSprint(sprintID)
}

func print(d *data) {
	fmt.Println("New..")
	for _, v := range d.new {
		fmt.Printf("%v, %v\n", v.Time, v.Value)
	}
	fmt.Println("Progress..")
	for _, v := range d.inProgress {
		fmt.Printf("%v, %v\n", v.Time, v.Value)
	}
}
func printTable(w io.Writer, header string, d []entry) {
	t := `
	<h2>{{ .Name }}</h2>
	<table>
	<tr><th>Time</th><th>Effort (secs)</th><th>Description</th></tr>
	{{ range .Entries }}
	<tr>
		<td>{{if .Time.IsZero }}before start of sprint{{else}}{{.Time }}{{end}}</td><td>{{ .Value }}</td><td>{{ .Msg }}</td>
	</tr>{{ end }}
	</table>`
	tpl, err := template.New("t").Parse(t)
	if err != nil {
		log.Fatal(err)
	}
	tpl.Execute(w, struct {
		Name    string
		Entries []entry
	}{header, d})
}
func printDiagram(w io.Writer, d diagram) {
	t := `
	<script type="text/javascript" src="https://www.gstatic.com/charts/loader.js"></script>
	   <div id="chart_div" style="width: 100%; height: 500px;"></div>
	   <p>Sprint start: {{ .Sprint.StartDate }}</p>
	   <p>Sprint end: {{ .Sprint.EndDate }}</p>
	   Generated: {{now}}
	<script>
	google.charts.load('current', {'packages':['corechart']});
	google.charts.setOnLoadCallback(drawChart);

	function drawChart() {
	var data = new google.visualization.DataTable();
	data.addColumn('date', 'Time');
	data.addColumn({type:'string', role:'annotation'});
	data.addColumn('number', 'New');
	data.addColumn('number', 'In Progress');
	data.addRows([
		{{ range .Entries }}
		[new Date(parseInt({{.Time.UnixNano }} /1000000)), null, {{ .New }}/3600,{{ .Progress}}/3600],{{ end }}
		{{ if .StartLine }}[new Date(parseInt({{ .Sprint.StartDate.UnixNano }} /1000000)), "Sprint start",null,null],{{end}}
		{{ if .Sprint.EndDate }}[new Date(parseInt({{ .Sprint.EndDate.UnixNano }} /1000000)), "Sprint end",null,null],{{end}}
		[null,null,null,null]
	]);

		var options = {
			title: 'Sprint Burndown (remaning effort) - {{ .Sprint.Name }}',
			hAxis: {title: 'Days',  titleTextStyle: {color: '#333'}},
			vAxis: {title: 'Hours remaining', minValue: 0},
			isStacked: true,
			annotations: {style:'line'}
		};

		var chart = new google.visualization.AreaChart(document.getElementById('chart_div'));
		chart.draw(data, options);
	}
	</script>`
	tpl, err := template.New("t").Funcs(template.FuncMap{"now": time.Now}).Parse(t)
	if err != nil {
		log.Fatal(err)
	}

	err = tpl.Execute(w, d)
	if err != nil {
		log.Fatal(err)
	}
}

func (d data) collapse(start time.Time) []tableEntry {
	n := dedupe(d.new)
	p := dedupe(d.inProgress)
	var result []tableEntry
	pi, ni := 0, 0
	var v tableEntry
	v.Time = time.Time{}
	for pi < len(p) || ni < len(n) {
		if pi < len(p) && v.Time == p[pi].Time {
			v.Progress = p[pi].Value
			pi++
		}
		if ni < len(n) && v.Time == n[ni].Time {
			v.New = n[ni].Value
			ni++
		}
		result = append(result, v)
		v = tableEntry{Time: nextTime(pi, ni, p, n), New: v.New, Progress: v.Progress}
	}
	if len(result) > 2 && result[0].Time.IsZero() {
		//Eliminate the 0 time
		result[0].Time = start
	}
	return result
}
func nextTime(pi, ni int, p, n []entry) time.Time {
	var t time.Time
	if pi < len(p) {
		t = p[pi].Time
		if ni < len(n) && n[ni].Time.Before(t) {
			return n[ni].Time
		}
		return t
	}
	if ni < len(n) {
		return n[ni].Time
	}
	return t
}

func dedupe(d []entry) []entry {
	progress := make([]entry, 0, len(d))
	sortByTime(d)
	for _, e := range d {
		if len(progress) == 0 {
			progress = append(progress, e)
		} else {
			lastItem := len(progress) - 1
			if progress[lastItem].Time == e.Time {
				progress[lastItem].Value += e.Value
			} else {
				e.Value += progress[lastItem].Value
				progress = append(progress, e)
			}
		}
	}
	return progress
}

func sortByTime(v []entry) {
	sort.Slice(v, func(i, j int) bool {
		return v[i].Time.Before(v[j].Time)
	})
}

func (d *data) collect(i jira.Issue) error {
	changes := getChangesAfter(i, d.start)
	lastStatus, lastEstimate := changes[0].newStatus, changes[0].newTime
	for _, change := range changes {
		//Find status at time index
		if change.statusChange {
			if change.timeChange {
				//status and estimate change
				if !change.time.IsZero() {
					d.addTimeEstimateChange(change.oldStatus, change.time, -change.oldTime, fmt.Sprintf("%s: change of status (from %s) and estimate", i.Key, change.oldStatus))
				}
				d.addTimeEstimateChange(change.newStatus, change.time, change.newTime, fmt.Sprintf("%s: updated status (to %s) and changed estimate", i.Key, change.newStatus))
				lastEstimate = change.newTime
			} else {
				//only status change
				if !change.time.IsZero() {
					d.addTimeEstimateChange(change.oldStatus, change.time, -lastEstimate, fmt.Sprintf("%s: change of status from %s", i.Key, change.oldStatus))
				}
				d.addTimeEstimateChange(change.newStatus, change.time, lastEstimate, fmt.Sprintf("%s: updated status to %s", i.Key, change.newStatus))
			}
			lastStatus = change.newStatus
		} else {
			//only estimate change
			estimate := change.newTime
			if !change.time.IsZero() {
				estimate -= change.oldTime
			}
			d.addTimeEstimateChange(lastStatus, change.time, estimate, fmt.Sprintf("%s: changed estimate", i.Key))
			lastEstimate = change.newTime
		}
	}
	return nil
}

//Get the changes after the give timestamp, at minimum gives the initial state with zero-time
func getChangesAfter(i jira.Issue, t time.Time) []change {
	changes := getStatusAndEstimateChanges(i.Changelog.Histories, t)

	//Find initial values
	var initialStatus string
	initialEstimate := -1
	for _, c := range changes {
		if c.statusChange && initialStatus == "" {
			initialStatus = c.oldStatus
		}
		if c.timeChange && initialEstimate == -1 {
			initialEstimate = c.oldTime
		}
		if initialStatus != "" && initialEstimate != -1 {
			break
		}
	}
	if initialStatus == "" {
		initialStatus = i.Fields.Status.Name
	}
	if initialEstimate == -1 {
		initialEstimate = i.Fields.TimeEstimate
	}

	return append([]change{change{time.Time{}, true, "", initialStatus, true, 0, initialEstimate}}, changes...)
}

type change struct {
	time         time.Time
	statusChange bool
	oldStatus    string
	newStatus    string
	timeChange   bool
	oldTime      int
	newTime      int
}

func getStatusAndEstimateChanges(histories []jira.ChangelogHistory, start time.Time) (result []change) {
	prevEstimate := -1
	for _, h := range histories {
		time, err := h.CreatedTime()
		if err != nil {
			log.Printf("Could not parse time from %s, ignoring history entry %v (by %v)", h.Created, h.Id, h.Author)
			continue
		}
		if time.Before(start) {
			continue
		}

		statusChange, oldState, newState, timeChange, oldEstimate, newEstimate := changedStateOrEstimate(h, prevEstimate)
		if statusChange || timeChange {
			result = append(result, change{time, statusChange, oldState, newState, timeChange, oldEstimate, newEstimate})
		}
		if timeChange {
			prevEstimate = newEstimate
		}
	}
	return
}
func changedStateOrEstimate(h jira.ChangelogHistory, prev int) (stateChanged bool, oldState string, newState string, timeChanged bool, oldValue int, newValue int) {
	for _, it := range h.Items {
		if it.Field == "timeestimate" {
			oldValue = parseInt(it.FromString)
			if prev != -1 && oldValue != prev {
				log.Printf("change history detail %s: weird state old estimates don't match, got %v, but expected previous %v. Using previous value instead.", h.Id, oldValue, prev)
				//Correct for inconsistencies
				oldValue = prev
			}
			newValue = parseInt(it.ToString)
			timeChanged = timeChanged || oldValue != newValue
		}
		if it.Field == "status" {
			oldState = it.FromString
			newState = it.ToString
			stateChanged = true
		}
	}
	return
}

// state, change time, change
func (d *data) addTimeEstimateChange(t string, time time.Time, diff int, msg string) {
	if diff == 0 || d.isDone(t) {
		return
	}
	var update *[]entry
	if d.isInProgress(t) {
		update = &d.inProgress
	} else {
		update = &d.new
	}
	updated := append(*update, entry{time, diff, msg})
	*update = updated
}

// is progress
func (d *data) isInProgress(state string) bool {
	return d.progressCategory[state]
}

// is done
func (d *data) isDone(state string) bool {
	return d.completeCategory[state]
}

func parseInt(s string) int {
	input := strings.TrimSpace(s)
	if input == "" || input == "null" {
		return 0
	}

	var err error
	value, err := strconv.Atoi(s)
	if err != nil {
		log.Print("Problem parsing to number", s, err)
	}
	return value
}
