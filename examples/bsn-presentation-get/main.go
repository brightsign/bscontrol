package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/brightsign/bscontrol"
	"github.com/brightsign/bscontrol/examples/internal/cli"
)

func main() {
	app := cli.New("Show one presentation, by ID or by name.",
		"By ID:\n    %s --id 42",
		"By name:\n    %s --name \"Lobby loop\"",
	)
	var (
		id   = flag.Int("id", 0, "Presentation ID")
		name = flag.String("name", "", "Presentation name")
	)
	app.Parse()

	if (*id > 0) == (*name != "") {
		app.Usagef("give exactly one of --id or --name")
	}

	client := app.Client()
	var (
		p   *bscontrol.Presentation
		err error
	)
	if *id > 0 {
		app.Statusf("Getting presentation %d...", *id)
		p, err = client.Presentations.GetByID(app.Context(), *id)
	} else {
		app.Statusf("Getting presentation %q...", *name)
		p, err = client.Presentations.GetByName(app.Context(), *name)
	}
	if err != nil {
		app.Fail("failed to get presentation: %v", err)
	}

	app.Output(p, func(w io.Writer) {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", p.ID, p.Status, p.Type, p.Name)
	})
}
