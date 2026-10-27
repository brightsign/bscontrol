package main

import (
	"flag"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/brightsign/bscontrol"
	"github.com/brightsign/bscontrol/examples/internal/cli"
)

const dateLayout = "2006-01-02"

func main() {
	app := cli.New("Schedule a presentation on a group.\n\nBy default the schedule is recurrent, all day, every day.",
		"All day, every day:\n    %s --group \"Lobby displays\" --presentation-name my-test",
		"Weekday mornings for a month:\n    %s --group G --presentation-id 42 --start 08:00:00 --duration 04:00:00 --days 62 --recurrence-start 2026-11-01 --recurrence-end 2026-11-30",
		"One-off event:\n    %s --group G --presentation-id 42 --recurrent=false --event-date 2026-12-24 --start 18:00:00 --duration 02:00:00",
	)
	var (
		group        = flag.String("group", "", "Group name (required)")
		presID       = flag.Int("presentation-id", 0, "Presentation ID")
		presName     = flag.String("presentation-name", "", "Presentation name (alternative to --presentation-id)")
		recurrent    = flag.Bool("recurrent", true, "Recurrent schedule (false: a single event on --event-date)")
		start        = flag.String("start", "00:00:00", "Start time of day, HH:MM:SS")
		duration     = flag.String("duration", "23:59:59", "Duration, HH:MM:SS")
		days         = flag.String("days", bscontrol.DaysOfWeekEveryDay, "Days of week: a name such as EveryDay, or a number")
		recStart     = flag.String("recurrence-start", "", "First day of a recurrent schedule, YYYY-MM-DD")
		recEnd       = flag.String("recurrence-end", "", "Last day of a recurrent schedule, YYYY-MM-DD")
		eventDate    = flag.String("event-date", "", "Day of a one-off event, YYYY-MM-DD")
		interruptSch = flag.Bool("interrupt", false, "Interrupt whatever is playing when this starts")
	)
	app.Parse()

	if *group == "" {
		app.Usagef("--group is required")
	}
	if (*presID > 0) == (*presName != "") {
		app.Usagef("give exactly one of --presentation-id or --presentation-name")
	}
	if !*recurrent && *eventDate == "" {
		app.Usagef("--event-date is required when --recurrent=false")
	}

	parseDate := func(flagName, v string) *time.Time {
		if v == "" {
			return nil
		}
		t, err := time.Parse(dateLayout, v)
		if err != nil {
			app.Usagef("--%s must be YYYY-MM-DD, got %q", flagName, v)
		}
		return &t
	}
	sched := &bscontrol.ScheduledPresentation{
		PresentationID:      *presID,
		IsRecurrent:         *recurrent,
		StartTime:           *start,
		Duration:            *duration,
		RecurrenceStartDate: parseDate("recurrence-start", *recStart),
		RecurrenceEndDate:   parseDate("recurrence-end", *recEnd),
		EventDate:           parseDate("event-date", *eventDate),
		InterruptScheduling: *interruptSch,
	}
	if n, err := strconv.Atoi(*days); err == nil {
		sched.DaysOfWeek = n
	} else {
		sched.DaysOfWeek = *days
	}

	client := app.Client()
	ctx := app.Context()

	if *presName != "" {
		app.Statusf("Looking up presentation %q...", *presName)
		p, err := client.Presentations.GetByName(ctx, *presName)
		if err != nil {
			app.Fail("failed to find presentation %q: %v", *presName, err)
		}
		sched.PresentationID, sched.PresentationName = p.ID, p.Name
	}

	app.Statusf("Scheduling presentation %d on group %q...", sched.PresentationID, *group)
	created, err := client.Schedules.AddScheduledPresentation(ctx, *group, sched)
	if err != nil {
		app.Fail("failed to add schedule: %v", err)
	}
	app.Statusf("Created schedule %d", created.ID)

	app.Output(created, func(w io.Writer) {
		fmt.Fprintf(w, "%d\tpresentation %d\t%s for %s\n", created.ID, created.PresentationID, created.StartTime, created.Duration)
	})
}
