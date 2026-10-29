package cli

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout runs fn and returns what it wrote to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()
	fn()
	w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestOutputJSONByDefault(t *testing.T) {
	a := &App{jsonOut: true}
	got := captureStdout(t, func() {
		a.Output(map[string]int{"id": 7}, func(w io.Writer) { io.WriteString(w, "human") })
	})
	if !strings.Contains(got, `"id": 7`) || strings.Contains(got, "human") {
		t.Errorf("stdout = %q, want JSON only", got)
	}
}

func TestOutputHumanWhenJSONOff(t *testing.T) {
	a := &App{jsonOut: false}
	got := captureStdout(t, func() {
		a.Output(map[string]int{"id": 7}, func(w io.Writer) { io.WriteString(w, "human\n") })
	})
	if got != "human\n" {
		t.Errorf("stdout = %q", got)
	}
}

func TestOutputFallsBackToJSONWithoutSummary(t *testing.T) {
	a := &App{jsonOut: false}
	got := captureStdout(t, func() { a.Output([]int{1}, nil) })
	if !strings.Contains(got, "1") {
		t.Errorf("stdout = %q", got)
	}
}

func TestConfirmSkipsPromptWithForce(t *testing.T) {
	a := &App{hasForce: true, force: true}
	if !a.Confirm("sure?") {
		t.Error("--force should answer yes without prompting")
	}
}

func TestConfirmReadsAnswerFromStdin(t *testing.T) {
	cases := map[string]bool{"y\n": true, "YES\n": true, " yes \n": true, "n\n": false, "\n": false, "maybe\n": false, "": false}
	for input, want := range cases {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.WriteString(input)
		w.Close()
		origIn := os.Stdin
		os.Stdin = r
		got := (&App{}).Confirm("sure?")
		os.Stdin = origIn
		if got != want {
			t.Errorf("Confirm with %q = %v, want %v", input, got, want)
		}
	}
}

// Without AddForce, --force does not exist, so an unset field must never skip
// the prompt even if it happens to be true.
func TestConfirmIgnoresForceWhenNotRegistered(t *testing.T) {
	r, w, _ := os.Pipe()
	w.Close()
	origIn := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origIn }()
	if (&App{force: true}).Confirm("sure?") {
		t.Error("force without hasForce must not auto-confirm")
	}
}
