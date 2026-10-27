package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/brightsign/bscontrol/examples/internal/cli"
)

func main() {
	app := cli.New("Delete presentations by ID, by name or by filter.\n\nA presentation that is still scheduled on a group has to be unscheduled first\n(see bsn-schedule-delete).",
		"Delete by name (asks first):\n    %s --name my-test",
		"Delete without asking:\n    %s --id 42 --force",
		"Delete the canary's leftovers:\n    %s --filter \"[Name] STARTS WITH 'content-test-'\" --force",
	)
	app.AddForce()
	var (
		id     = flag.Int("id", 0, "Presentation ID to delete")
		name   = flag.String("name", "", "Presentation name to delete")
		filter = flag.String("filter", "", "Filter expression: delete every presentation it matches")
	)
	app.Parse()

	given := 0
	if *id > 0 {
		given++
	}
	if *name != "" {
		given++
	}
	if *filter != "" {
		given++
	}
	if given != 1 {
		app.Usagef("give exactly one of --id, --name or --filter")
	}

	client := app.Client()
	ctx := app.Context()

	switch {
	case *id > 0:
		if !app.Confirm(fmt.Sprintf("Delete presentation %d?", *id)) {
			app.Fail("cancelled")
		}
		if err := client.Presentations.DeleteByID(ctx, *id); err != nil {
			app.Fail("failed to delete presentation %d: %v", *id, err)
		}
		app.Statusf("Deleted presentation %d", *id)
		app.Output(map[string]any{"deleted": true, "id": *id}, func(w io.Writer) { fmt.Fprintf(w, "deleted %d\n", *id) })
	case *name != "":
		if !app.Confirm(fmt.Sprintf("Delete presentation %q?", *name)) {
			app.Fail("cancelled")
		}
		if err := client.Presentations.DeleteByName(ctx, *name); err != nil {
			app.Fail("failed to delete presentation %q: %v", *name, err)
		}
		app.Statusf("Deleted presentation %q", *name)
		app.Output(map[string]any{"deleted": true, "name": *name}, func(w io.Writer) { fmt.Fprintf(w, "deleted %s\n", *name) })
	default:
		if !app.Confirm(fmt.Sprintf("Delete ALL presentations matching %q?", *filter)) {
			app.Fail("cancelled")
		}
		res, err := client.Presentations.DeleteByFilter(ctx, *filter)
		if err != nil {
			app.Fail("failed to delete presentations: %v", err)
		}
		app.Statusf("Deleted %d presentation(s)", res.DeletedCount)
		app.Output(res, func(w io.Writer) { fmt.Fprintf(w, "deleted %d\n", res.DeletedCount) })
	}
}
