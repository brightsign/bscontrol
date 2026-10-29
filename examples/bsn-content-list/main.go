package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/brightsign/bscontrol"
	"github.com/brightsign/bscontrol/examples/internal/cli"
)

func main() {
	app := cli.New("List content in the network's content library.",
		"List everything:\n    %s --all",
		"List one folder:\n    %s --folder videos/promos",
		"Filter by name:\n    %s --filter \"[FileName] IS 'clip.mp4'\" --all",
	)
	var (
		folder   = flag.String("folder", "", "List only this virtual folder (e.g. videos/promos); default is the whole library")
		filter   = flag.String("filter", "", "Filter expression (use '' to double a quote inside a value)")
		sortExpr = flag.String("sort", "", "Sort expression (e.g. fileName)")
		pageSize = flag.Int("page-size", 100, "Items per page")
		marker   = flag.String("marker", "", "Continue from this pagination marker")
		all      = flag.Bool("all", false, "Follow pagination and return every item (not with --folder)")
	)
	app.Parse()

	if *all && *folder != "" {
		app.Usagef("--all and --folder cannot be combined; page a folder with --marker")
	}

	client := app.Client()
	ctx := app.Context()

	var opts []bscontrol.ListOption
	if *pageSize > 0 {
		opts = append(opts, bscontrol.WithPageSize(*pageSize))
	}
	if *filter != "" {
		opts = append(opts, bscontrol.WithFilter(*filter))
	}
	if *sortExpr != "" {
		opts = append(opts, bscontrol.WithSort(*sortExpr))
	}
	if *marker != "" {
		opts = append(opts, bscontrol.WithMarker(*marker))
	}

	var result bscontrol.ContentFileList
	switch {
	case *all:
		app.Statusf("Listing all content...")
		items, err := client.Content.ListAll(ctx, opts...)
		if err != nil {
			app.Fail("failed to list content: %v", err)
		}
		result = bscontrol.ContentFileList{Items: items, TotalCount: len(items)}
	case *folder != "":
		app.Statusf("Listing folder %q...", *folder)
		page, err := client.Content.ListFolder(ctx, *folder, opts...)
		if err != nil {
			app.Fail("failed to list folder: %v", err)
		}
		result = *page
	default:
		app.Statusf("Listing content (one page)...")
		page, err := client.Content.List(ctx, opts...)
		if err != nil {
			app.Fail("failed to list content: %v", err)
		}
		result = *page
	}

	app.Statusf("%d item(s)", len(result.Items))
	if result.IsTruncated {
		app.Statusf("More results: re-run with --marker %q, or use --all", result.NextMarker)
	}
	app.Output(result, func(w io.Writer) {
		for _, f := range result.Items {
			fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%s\n", f.ID, f.MediaType, f.VirtualPath, f.FileSize, f.FileName)
		}
	})
}
