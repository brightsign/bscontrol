package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/brightsign/bscontrol"
	"github.com/brightsign/bscontrol/examples/internal/cli"
)

func main() {
	app := cli.New("Upload a file to the network's content library.",
		"Upload to the root:\n    %s --file clip.mp4",
		"Upload into a folder:\n    %s --file clip.mp4 --path videos/promos",
		"Force a media type:\n    %s --file data.bin --media-type Auxiliary",
	)
	var (
		file      = flag.String("file", "", "Local file to upload (required)")
		path      = flag.String("path", "", "Virtual folder in the library, either slash style (default: the root)")
		mediaType = flag.String("media-type", "", "Override the media type (Image, Video, Audio, Text, Font, Stylesheet, Auxiliary, ...); default is detected from the extension")
	)
	app.Parse()

	if *file == "" {
		app.Usagef("--file is required")
	}

	client := app.Client()

	opts := []bscontrol.UploadOption{
		bscontrol.WithUploadProgress(func(p bscontrol.UploadProgress) {
			app.Debugf("  sent %d / %d bytes", p.Sent, p.Total)
		}),
	}
	if *mediaType != "" {
		opts = append(opts, bscontrol.WithUploadMediaType(*mediaType))
	}

	app.Statusf("Uploading %s...", *file)
	res, err := client.Upload.Upload(app.Context(), *file, *path, opts...)
	if err != nil {
		app.Fail("upload failed: %v", err)
	}
	if res.Deduplicated {
		app.Statusf("Already in the library: reusing content %d (no bytes sent)", res.ContentID)
	} else {
		app.Statusf("Uploaded as content %d", res.ContentID)
	}

	app.Output(res, func(w io.Writer) {
		fmt.Fprintf(w, "%d\t%s\t%s\n", res.ContentID, res.VirtualPath, res.FileName)
	})
}
