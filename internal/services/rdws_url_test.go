package services

import (
	"fmt"
	"strings"
	"testing"
)

func TestEscQueryNeutralizesInjection(t *testing.T) {
	// A serial carrying query metacharacters must not be able to add a second
	// destinationName (which would redirect the request to another player).
	malicious := "VICTIM&destinationName=ATTACKER"
	got := escQuery(malicious)
	if strings.Contains(got, "&destinationName=ATTACKER") {
		t.Fatalf("escQuery left an injectable &destinationName: %q", got)
	}

	url := fmt.Sprintf("%s/info/?destinationType=player&destinationName=%s", "https://ws.example", escQuery(malicious))
	// The only destinationName= in the final URL must be the one we wrote.
	if n := strings.Count(url, "destinationName="); n != 1 {
		t.Fatalf("expected exactly one destinationName= in %q, got %d", url, n)
	}
}

func TestEscQueryIsNoOpForValidSerial(t *testing.T) {
	serial := "XD1A2B3C4D5E"
	if got := escQuery(serial); got != serial {
		t.Errorf("escQuery(%q) = %q, want unchanged", serial, got)
	}
}

func TestEscSegmentEncodesSlashAndSpace(t *testing.T) {
	// A single path segment has no slashes of its own, so an embedded slash must
	// be encoded rather than create a new segment.
	if got := escSegment("a/b c"); got != "a%2Fb%20c" {
		t.Errorf("escSegment(\"a/b c\") = %q, want %q", got, "a%2Fb%20c")
	}
}

func TestEscPathPreservesSeparatorsButEncodesSegments(t *testing.T) {
	// A device path keeps its slash separators and leading slash, but each
	// segment is encoded so a space or query metacharacter cannot break out.
	got := escPath("/storage/sd/my file?x")
	want := "/storage/sd/my%20file%3Fx"
	if got != want {
		t.Errorf("escPath = %q, want %q", got, want)
	}
}
