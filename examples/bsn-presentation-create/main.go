package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/brightsign/bscontrol"
	"github.com/brightsign/bscontrol/examples/internal/cli"
)

func main() {
	app := cli.New("Create a presentation.\n\nWith only --name you get an empty presentation shell. For a real one (project\nfile, media, autoplay), pass the full request as JSON with --from-json; any of\n--name, --device-model, --language and --status given alongside it override the file.",
		"An empty presentation:\n    %s --name my-test --device-model XT1144 --language English",
		"From a request file:\n    %s --from-json request.json --name my-test-2",
	)
	var (
		name        = flag.String("name", "", "Presentation name (required unless the JSON file has one)")
		fromJSON    = flag.String("from-json", "", "File holding a PresentationCreateRequest as JSON")
		deviceModel = flag.String("device-model", "", "Player model the presentation targets (e.g. XT1144)")
		language    = flag.String("language", "", "Language (e.g. English)")
		status      = flag.String("status", "", "Status (e.g. Draft, Published)")
	)
	app.Parse()

	var req bscontrol.PresentationCreateRequest
	if *fromJSON != "" {
		data, err := os.ReadFile(*fromJSON)
		if err != nil {
			app.Fail("cannot read %s: %v", *fromJSON, err)
		}
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields() // a typo'd key should fail loudly, not be dropped
		if err := dec.Decode(&req); err != nil {
			app.Fail("%s is not a valid presentation request: %v", *fromJSON, err)
		}
	}
	if *name != "" {
		req.Name = *name
	}
	if *deviceModel != "" {
		req.DeviceModel = *deviceModel
	}
	if *language != "" {
		req.Language = *language
	}
	if *status != "" {
		req.Status = *status
	}
	if req.Name == "" {
		app.Usagef("a presentation name is required (--name or in the JSON file)")
	}

	client := app.Client()
	app.Statusf("Creating presentation %q...", req.Name)
	created, err := client.Presentations.Create(app.Context(), &req)
	if err != nil {
		app.Fail("failed to create presentation: %v", err)
	}
	app.Statusf("Created presentation %d", created.ID)

	app.Output(created, func(w io.Writer) {
		fmt.Fprintf(w, "%d\t%s\t%s\n", created.ID, created.Status, created.Name)
	})
}
