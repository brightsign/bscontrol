// Package cli holds the small amount of plumbing the content, presentation and
// schedule example tools share, so each main.go can stay about the one call it
// demonstrates.
//
// Conventions every tool built on it follows:
//   - credentials come from BS_CLIENT_ID / BS_SECRET, the network from --network
//     (or -n) falling back to BS_NETWORK
//   - status and progress go to stderr, the result goes to stdout
//   - stdout is JSON by default; --json=false prints a short human summary
//   - exit code 0 on success, 1 on any error
package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/brightsign/bscontrol"
)

// App is one running tool.
type App struct {
	network  string
	timeout  int
	verbose  bool
	jsonOut  bool
	help     bool
	force    bool
	hasForce bool

	ctx    context.Context
	cancel context.CancelFunc
}

// New registers the common flags and the usage text. Register the tool's own
// flags on the flag package after calling it, then call Parse.
func New(summary string, examples ...string) *App {
	a := &App{}
	flag.BoolVar(&a.help, "help", false, "Display usage information")
	flag.StringVar(&a.network, "network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(&a.network, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")
	flag.IntVar(&a.timeout, "timeout", 30, "HTTP request timeout in seconds")
	flag.BoolVar(&a.verbose, "verbose", false, "Show extra details on stderr")
	flag.BoolVar(&a.jsonOut, "json", true, "Write the result as JSON to stdout (--json=false for a short summary)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n%s\n\nOptions:\n", os.Args[0], summary)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID       BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (required unless --network is given)\n")
		if len(examples) > 0 {
			fmt.Fprintf(os.Stderr, "\nExamples:\n")
			for _, ex := range examples {
				fmt.Fprintf(os.Stderr, "  %s\n", strings.ReplaceAll(ex, "%s", os.Args[0]))
			}
		}
	}
	return a
}

// AddForce registers -y / --force for destructive tools.
func (a *App) AddForce() {
	a.hasForce = true
	flag.BoolVar(&a.force, "force", false, "Skip the confirmation prompt")
	flag.BoolVar(&a.force, "y", false, "Skip the confirmation prompt [alias for --force]")
}

// Parse reads the flags and handles --help.
func (a *App) Parse() {
	flag.Parse()
	if a.help {
		flag.Usage()
		os.Exit(0)
	}
	a.ctx, a.cancel = signal.NotifyContext(context.Background(), os.Interrupt)
}

// Context is cancelled on Ctrl-C.
func (a *App) Context() context.Context { return a.ctx }

// Verbose reports whether --verbose was given.
func (a *App) Verbose() bool { return a.verbose }

// Statusf writes a progress line to stderr.
func (a *App) Statusf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// Debugf writes to stderr only with --verbose.
func (a *App) Debugf(format string, args ...any) {
	if a.verbose {
		a.Statusf(format, args...)
	}
}

// Fail prints an error to stderr and exits 1.
func (a *App) Fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

// Usagef reports a bad invocation (a missing required flag) and exits 1.
func (a *App) Usagef(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n\n", args...)
	flag.Usage()
	os.Exit(1)
}

// Client builds the SDK client, authenticates and selects the network.
func (a *App) Client() *bscontrol.Client {
	opts := []bscontrol.Option{bscontrol.WithEnv(), bscontrol.WithTimeout(time.Duration(a.timeout) * time.Second)}
	if a.network != "" {
		opts = append(opts, bscontrol.WithNetwork(a.network))
	}
	client, err := bscontrol.New(opts...)
	if err != nil {
		if bscontrol.IsConfigurationError(err) {
			a.Fail("configuration: %v", err)
		}
		a.Fail("failed to create client: %v", err)
	}
	if client.Config().NetworkName == "" {
		a.Fail("no network selected: pass --network or set BS_NETWORK")
	}

	a.Debugf("Authenticating with BSN.cloud...")
	if err := client.EnsureReady(a.ctx); err != nil {
		if bscontrol.IsAuthenticationError(err) {
			a.Fail("authentication failed: %v", err)
		}
		a.Fail("%v", err)
	}
	a.Debugf("Using network %q", client.Config().NetworkName)
	return client
}

// Output writes the result: JSON to stdout by default, or, with --json=false,
// whatever human prints.
func (a *App) Output(v any, human func(w io.Writer)) {
	if a.jsonOut || human == nil {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(v); err != nil {
			a.Fail("failed to write JSON: %v", err)
		}
		return
	}
	human(os.Stdout)
}

// Confirm asks a yes/no question on stderr unless -y/--force was given (see
// AddForce). Anything but y or yes is a no.
func (a *App) Confirm(prompt string) bool {
	if a.hasForce && a.force {
		return true
	}
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", prompt)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	}
	return false
}
