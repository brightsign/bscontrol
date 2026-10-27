package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/brightsign/bscontrol/examples/internal/cli"
)

func main() {
	app := cli.New("List the presentations scheduled on a group.",
		"List a group's schedule:\n    %s --group \"Lobby displays\"",
	)
	group := flag.String("group", "", "Group name (required)")
	app.Parse()

	if *group == "" {
		app.Usagef("--group is required")
	}

	client := app.Client()
	app.Statusf("Getting schedule for group %q...", *group)
	schedule, err := client.Schedules.GetGroupSchedule(app.Context(), *group)
	if err != nil {
		app.Fail("failed to get schedule: %v", err)
	}
	app.Statusf("%d scheduled presentation(s)", len(schedule.Items))

	app.Output(schedule, func(w io.Writer) {
		for _, s := range schedule.Items {
			fmt.Fprintf(w, "%d\tpresentation %d\t%s\t%s for %s\trecurrent=%t\n",
				s.ID, s.PresentationID, s.PresentationName, s.StartTime, s.Duration, s.IsRecurrent)
		}
	})
}
