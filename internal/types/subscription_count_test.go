package types

import (
	"encoding/json"
	"testing"
)

// The BSN.cloud GET /Subscriptions/Count endpoint returns a bare JSON number
// (observed: 158), which is what the SDK must decode. A struct decode of a bare
// number fails, so SubscriptionCount must accept the bare-number form.
func TestSubscriptionCount_UnmarshalBareNumber(t *testing.T) {
	var count SubscriptionCount
	if err := json.Unmarshal([]byte("158"), &count); err != nil {
		t.Fatalf("unmarshal bare number: %v", err)
	}
	if count.Count != 158 {
		t.Errorf("Count = %d, want 158", count.Count)
	}
}

// Some Count endpoints in the API are documented to return an object form
// ({"count": N}) instead. SubscriptionCount must keep decoding that shape too,
// so the bare-number support does not regress the object form.
func TestSubscriptionCount_UnmarshalObject(t *testing.T) {
	var count SubscriptionCount
	if err := json.Unmarshal([]byte(`{"count": 42}`), &count); err != nil {
		t.Fatalf("unmarshal object: %v", err)
	}
	if count.Count != 42 {
		t.Errorf("Count = %d, want 42", count.Count)
	}
}

// A body that is neither a number nor the expected object is a contract
// violation and must surface as an error, not a silent zero.
func TestSubscriptionCount_UnmarshalUnexpectedShapeErrors(t *testing.T) {
	var count SubscriptionCount
	if err := json.Unmarshal([]byte(`"not-a-count"`), &count); err == nil {
		t.Error("expected error unmarshaling a string body, got nil")
	}
}

// A JSON null body must not silently decode as Count: 0 — unmarshaling null
// into a plain int is a documented Go no-op, which would make a genuinely
// unexpected null response indistinguishable from "the network has zero
// subscriptions."
func TestSubscriptionCount_UnmarshalNullErrors(t *testing.T) {
	var count SubscriptionCount
	if err := json.Unmarshal([]byte("null"), &count); err == nil {
		t.Error("expected error unmarshaling a null body, got nil")
	}
}
