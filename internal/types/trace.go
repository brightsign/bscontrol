package types

import "time"

// RequestTrace is a timing-and-outcome record for a single HTTP request the SDK
// made. One is delivered to a tracer registered with WithRequestTracer after the
// request completes, including after any retries.
//
// Duration fields are measured by the transport. Ones it did not measure for a
// given request are zero: on a reused connection DNSLookup and TLSHandshake are
// zero because no lookup or handshake happened, and on a request that failed
// before a response StatusCode and BodyBytes are zero while Err is set.
type RequestTrace struct {
	Method       string        // HTTP method
	URL          string        // request URL
	StatusCode   int           // HTTP status code, or 0 if no response was received
	BodyBytes    int64         // response body size in bytes
	Attempt      int           // request attempts made, including retries (1 = no retry)
	DNSLookup    time.Duration // DNS resolution time; 0 on a reused connection
	TLSHandshake time.Duration // TLS handshake time; 0 on a reused connection
	ServerTime   time.Duration // time from the request being written to the first response byte
	TotalTime    time.Duration // full end-to-end round trip, including retries
	ConnReused   bool          // the connection was reused from an earlier request
	Err          error         // non-nil if the request ultimately failed
}

// RequestTracer receives one RequestTrace per request the SDK completes. It runs
// synchronously on the SDK's request goroutine, so it must not block; hand work
// to another goroutine or a buffered channel if the sink can be slow.
type RequestTracer func(RequestTrace)
