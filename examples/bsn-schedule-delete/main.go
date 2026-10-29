package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/brightsign/bscontrol/examples/internal/cli"
)

func main() {
	app := cli.New("Remove one scheduled presentation from a group.\n\nGet the schedule ID from bsn-schedule-list.",
		"Remove schedule 31 (asks first):\n    %s --group \"Lobby displays\" --id 31",
		"Remove without asking:\n    %s --group G --id 31 --force",
	)
	app.AddForce()
	var (
		group = flag.String("group", "", "Group name (required)")
		id    = flag.Int("id", 0, "Schedule ID (required)")
	)
	app.Parse()

	if *group == "" {
		app.Usagef("--group is required")
	}
	if *id <= 0 {
		app.Usagef("--id is required")
	}

	client := app.Client()
	if !app.Confirm(fmt.Sprintf("Remove schedule %d from group %q?", *id, *group)) {
		app.Fail("cancelled")
	}
	if err := client.Schedules.DeleteScheduledPresentation(app.Context(), *group, *id); err != nil {
		app.Fail("failed to remove schedule %d: %v", *id, err)
	}
	app.Statusf("Removed schedule %d from group %q", *id, *group)

	app.Output(map[string]any{"deleted": true, "group": *group, "id": *id},
		func(w io.Writer) { fmt.Fprintf(w, "deleted %d\n", *id) })
}
