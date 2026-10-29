package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/brightsign/bscontrol/examples/internal/cli"
)

func main() {
	app := cli.New("Download a content file (or its thumbnail) from the library.",
		"Save to a file:\n    %s --id 12345 --output clip.mp4",
		"Stream to stdout:\n    %s --id 12345 --output - > clip.mp4",
		"Save the thumbnail:\n    %s --id 12345 --thumbnail --max-width 320 --output thumb.png",
	)
	var (
		id        = flag.Int("id", 0, "Content ID (required)")
		output    = flag.String("output", "", "File to write, or - for stdout (required)")
		thumbnail = flag.Bool("thumbnail", false, "Fetch the server-generated thumbnail instead of the file")
		maxWidth  = flag.Int("max-width", 0, "Thumbnail max width (default: server's 100)")
		maxHeight = flag.Int("max-height", 0, "Thumbnail max height (default: server's 100)")
		overwrite = flag.Bool("overwrite", false, "Replace --output if it already exists")
	)
	app.Parse()

	if *id <= 0 {
		app.Usagef("--id is required")
	}
	if *output == "" {
		app.Usagef("--output is required")
	}

	client := app.Client()
	ctx := app.Context()

	// With --output - the bytes own stdout, so no JSON summary is written there.
	toStdout := *output == "-"
	var out io.Writer = os.Stdout
	var f *os.File
	if !toStdout {
		flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
		if *overwrite {
			flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		}
		var err error
		if f, err = os.OpenFile(*output, flags, 0o644); err != nil {
			app.Fail("cannot create %s: %v (use --overwrite to replace it)", *output, err)
		}
		out = f
	}
	fail := func(format string, args ...any) {
		if f != nil {
			f.Close()
			os.Remove(*output) // do not leave a partial file behind
		}
		app.Fail(format, args...)
	}

	var written int64
	if *thumbnail {
		app.Statusf("Fetching thumbnail for content %d...", *id)
		data, err := client.Content.GetThumbnail(ctx, *id, *maxWidth, *maxHeight)
		if err != nil {
			fail("failed to get thumbnail: %v", err)
		}
		n, err := out.Write(data)
		if err != nil {
			fail("failed to write output: %v", err)
		}
		written = int64(n)
	} else {
		app.Statusf("Downloading content %d...", *id)
		n, err := client.Content.DownloadTo(ctx, *id, out)
		if err != nil {
			fail("failed to download content %d: %v", *id, err)
		}
		written = n
	}
	if f != nil {
		if err := f.Close(); err != nil {
			app.Fail("failed to finish %s: %v", *output, err)
		}
	}
	app.Statusf("Wrote %d bytes", written)

	if !toStdout {
		app.Output(map[string]any{"id": *id, "output": *output, "bytes": written, "thumbnail": *thumbnail},
			func(w io.Writer) { fmt.Fprintf(w, "%s\t%d\n", *output, written) })
	}
}
