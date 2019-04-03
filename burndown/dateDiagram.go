package burndown

import (
	"html/template"
	"io"
	"log"
	"time"

	jira "gopkg.in/andygrunwald/go-jira.v1"
)

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

func (d data) prepareDiagram(s jira.Sprint, startTime time.Time, startMargin bool) diagram {
	sum := d.collapse(startTime)
	return diagram{s, sum, startMargin}
}

func (d diagram) printDiagram(w io.Writer) {
	t := `
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
