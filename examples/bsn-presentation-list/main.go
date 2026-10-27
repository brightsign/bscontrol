package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/brightsign/bscontrol"
	"github.com/brightsign/bscontrol/examples/internal/cli"
)

func main() {
	app := cli.New("List presentations on the network.",
		"List every presentation:\n    %s --all",
		"Only the canary's test presentations:\n    %s --filter \"[Name] STARTS WITH 'content-test-'\" --all",
	)
	var (
		filter   = flag.String("filter", "", "Filter expression (use '' to double a quote inside a value)")
		sortExpr = flag.String("sort", "", "Sort expression (e.g. name)")
		pageSize = flag.Int("page-size", 100, "Items per page")
		marker   = flag.String("marker", "", "Continue from this pagination marker")
		all      = flag.Bool("all", false, "Follow pagination and return every presentation")
	)
	app.Parse()

	client := app.Client()

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

	app.Statusf("Listing presentations...")
	var result bscontrol.PresentationList
	if *all {
		items, err := client.Presentations.ListAll(app.Context(), opts...)
		if err != nil {
			app.Fail("failed to list presentations: %v", err)
		}
		result = bscontrol.PresentationList{Items: items, TotalCount: len(items)}
	} else {
		page, err := client.Presentations.List(app.Context(), opts...)
		if err != nil {
			app.Fail("failed to list presentations: %v", err)
		}
		result = *page
	}

	app.Statusf("%d presentation(s)", len(result.Items))
	if result.IsTruncated {
		app.Statusf("More results: re-run with --marker %q, or use --all", result.NextMarker)
	}
	app.Output(result, func(w io.Writer) {
		for _, p := range result.Items {
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", p.ID, p.Status, p.Type, p.Name)
		}
	})
}
