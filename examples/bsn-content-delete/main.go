package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/brightsign/bscontrol/examples/internal/cli"
)

func main() {
	app := cli.New("Delete content from the library, by ID or by filter.",
		"Delete one item (asks first):\n    %s --id 12345",
		"Delete without asking:\n    %s --id 12345 --force",
		"Delete everything matching a name:\n    %s --filter \"[FileName] IS 'clip.mp4'\" --force",
	)
	app.AddForce()
	var (
		id     = flag.Int("id", 0, "Content ID to delete")
		filter = flag.String("filter", "", "Filter expression: delete every item it matches")
	)
	app.Parse()

	if (*id > 0) == (*filter != "") {
		app.Usagef("give exactly one of --id or --filter")
	}

	client := app.Client()
	ctx := app.Context()

	if *id > 0 {
		if !app.Confirm(fmt.Sprintf("Delete content %d?", *id)) {
			app.Fail("cancelled")
		}
		if err := client.Content.DeleteByID(ctx, *id); err != nil {
			app.Fail("failed to delete content %d: %v", *id, err)
		}
		app.Statusf("Deleted content %d", *id)
		app.Output(map[string]any{"deleted": true, "id": *id}, func(w io.Writer) { fmt.Fprintf(w, "deleted %d\n", *id) })
		return
	}

	if !app.Confirm(fmt.Sprintf("Delete ALL content matching %q?", *filter)) {
		app.Fail("cancelled")
	}
	res, err := client.Content.Delete(ctx, *filter)
	if err != nil {
		app.Fail("failed to delete content: %v", err)
	}
	app.Statusf("Deleted %d item(s)", res.DeletedCount)
	app.Output(res, func(w io.Writer) { fmt.Fprintf(w, "deleted %d\n", res.DeletedCount) })
}
