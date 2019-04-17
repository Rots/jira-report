package burndown

import (
	"fmt"
	"html/template"
	"io"
	"log"
	info "reports/jira"
	"time"

	jira "gopkg.in/andygrunwald/go-jira.v1"
)

type hoursDiagram struct {
	Sprint    jira.Sprint
	Entries   []sprintHoursEntry
	StartLine bool
	WorkInfo  converter
}
type sprintHoursEntry struct {
	Time     time.Duration
	New      int
	Progress int
}

//Converter converts timestamps to sprint working time (duration from sprint start)
type converter struct {
	info.BoardInfo
	Start                              time.Time
	WorkDayStartHours, WorkDayEndHours int
}

func (d data) prepareWorkHoursDiagram(s jira.Sprint, startTime time.Time, startMargin bool, workInfo info.BoardInfo, workdayStart, workdayEnd int) hoursDiagram {
	sum := d.collapse(startTime)
	conv := converter{
		BoardInfo:         workInfo,
		Start:             *s.StartDate,
		WorkDayStartHours: workdayStart,
		WorkDayEndHours:   workdayEnd,
	}
	e := conv.convertToSprintHoursEntries(sum)
	//As conversion may have created duplicate entries for the same time, eliminate these
	e = dedupeHours(e)
	return hoursDiagram{s, e, startMargin, conv}
}

func dedupeHours(e []sprintHoursEntry) []sprintHoursEntry {
	result := make([]sprintHoursEntry, 0, len(e))
	for _, v := range e {
		if len(result) > 0 && result[len(result)-1].Time == v.Time {
			//Only keep the last entry for that time
			result[len(result)-1] = v
		} else {
			result = append(result, v)
		}
	}
	return result
}

func (hd converter) convertToSprintHoursEntries(e []tableEntry) []sprintHoursEntry {
	result := make([]sprintHoursEntry, 0, len(e))
	for _, v := range e {
		result = append(result, sprintHoursEntry{
			Time:     hd.toSprintWorkTime(hd.Start, v.Time),
			New:      v.New,
			Progress: v.Progress,
		})
	}
	return result
}

func (d hoursDiagram) printDiagram(w io.Writer) {
	t := `
	   <div id="workHours" style="width: 100%; height: 500px;"></div>
	   <p>Sprint start: {{ .Sprint.StartDate }}</p>
	   <p>Sprint end: {{ .Sprint.EndDate }}</p>
	   <p>Total hours in Sprint: {{ convSprintWorkHours .Sprint.EndDate }}</p>
	   <p>Work Hours: {{ workDayHours }}</p>
	   Generated: {{now}}
	<script>
	google.charts.load('current', {'packages':['corechart']});
	google.charts.setOnLoadCallback(drawChart);

	function drawChart() {
	var data = new google.visualization.DataTable();
	data.addColumn('number', 'Time');
	data.addColumn({type:'string', role:'annotation'});
	data.addColumn('number', 'New');
	data.addColumn('number', 'In Progress');
	data.addRows([
		{{ range .Entries }}
		[{{ SprintWorkHours .Time }}, null, {{ .New }}/3600,{{ .Progress}}/3600],{{ end }}
		{{ if .StartLine }}[0, "Sprint start",null,null],{{end}}
		{{ if .Sprint.EndDate }}[{{ convSprintWorkHours .Sprint.EndDate }}, "Sprint end",null,null],{{end}}
		[null,null,null,null]
	]);

		var options = {
			title: 'Sprint Burndown (remaning effort) - {{ .Sprint.Name }}',
			hAxis: {title: 'Sprint work hours',  titleTextStyle: {color: '#333'}},
			vAxis: {title: 'Hours remaining', minValue: 0},
			isStacked: true,
			annotations: {style:'line'}
		};

		var chart = new google.visualization.AreaChart(document.getElementById('workHours'));
		chart.draw(data, options);
	}
	</script>`
	tpl, err := template.New("t").Funcs(template.FuncMap{
		"now": time.Now,
		"SprintWorkHours": func(t time.Duration) float64 {
			return t.Seconds() / 3600
		},
		"convSprintWorkHours": func(t time.Time) float64 {
			return d.WorkInfo.toSprintWorkTime(*d.Sprint.StartDate, t).Seconds() / 3600
		},
		// "nice": func(secs float32) string {
		// 	return fmt.Sprintf("%.2f", secs/3600)
		// },
		"workDayHours": func() string {
			return fmt.Sprintf("%02d:00 - %02d:00", d.WorkInfo.WorkDayStartHours, d.WorkInfo.WorkDayEndHours)
		},
	}).Parse(t)
	if err != nil {
		log.Fatal(err)
	}

	err = tpl.Execute(w, d)
	if err != nil {
		log.Fatal(err)
	}
}

func (hd converter) toSprintWorkTime(start, t time.Time) time.Duration {
	isNegativeMultiplier := int64(1)
	if t.Before(start) {
		isNegativeMultiplier = -1
		//Swap
		start, t = t, start
	}
	//Simulate workdays
	currentTime, t := hd.toWorkTime(start), hd.toWorkTime(t)
	currentDuration := int64(0)

	for currentTime.Before(t) {
		if isSameDay(currentTime, t) {
			currentDuration += int64(t.Sub(currentTime))
			currentTime = t
		} else {
			y, m, d := currentTime.Date()
			if hd.isWorkDay(currentTime) {
				currentDuration += int64(time.Date(y, m, d, hd.WorkDayEndHours, 0, 0, 0, currentTime.Location()).Sub(currentTime))
			}
			currentTime = time.Date(y, m, d+1, hd.WorkDayStartHours, 0, 0, 0, currentTime.Location())
		}
	}
	return time.Duration(currentDuration * isNegativeMultiplier)
}
func (hd converter) toWorkTime(t time.Time) time.Time {
	for !hd.isWorkTime(t) {
		//get to the first working date at start time
		y, m, d := t.Date()
		if t.Hour() < hd.WorkDayStartHours {
			t = time.Date(y, m, d, hd.WorkDayStartHours, 0, 0, 0, t.Location())
		} else {
			t = time.Date(y, m, d+1, hd.WorkDayStartHours, 0, 0, 0, t.Location())
		}
	}
	return t
}

func (hd converter) isWorkTime(t time.Time) bool {
	if !hd.isWorkDay(t) {
		return false
	}
	workTime := t.Hour() >= hd.WorkDayStartHours && t.Hour() < hd.WorkDayEndHours
	return workTime
}
func (hd converter) isWorkDay(t time.Time) bool {
	return hd.WeekDays[t.Weekday()] && !containsDate(hd.NonWorkingDays, t)
}
func containsDate(collection []time.Time, t time.Time) bool {
	for _, v := range collection {
		if isSameDay(v, t) {
			return true
		}
	}
	return false
}
func isSameDay(a, b time.Time) bool {
	y, m, d := a.Date()
	ty, tm, td := b.Date()
	return y == ty && m == tm && d == td
}
