package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/brightsign/bscontrol/examples/internal/cli"
)

func main() {
	app := cli.New("Show one content item's metadata.",
		"Get content 12345:\n    %s --id 12345",
		"One-line summary:\n    %s --id 12345 --json=false",
	)
	id := flag.Int("id", 0, "Content ID (required)")
	app.Parse()

	if *id <= 0 {
		app.Usagef("--id is required")
	}

	client := app.Client()
	app.Statusf("Getting content %d...", *id)
	file, err := client.Content.GetByID(app.Context(), *id)
	if err != nil {
		app.Fail("failed to get content %d: %v", *id, err)
	}

	app.Output(file, func(w io.Writer) {
		fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%s\n", file.ID, file.MediaType, file.VirtualPath, file.FileSize, file.FileName)
	})
}
