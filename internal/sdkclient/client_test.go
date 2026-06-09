package sdkclient

import (
	"errors"
	"testing"
)

// TestIsResourceGone locks in the gone-detection contract used by per-environment
// resource Read/Delete handlers. A 404 always counts as gone (unchanged legacy
// behavior). The manager's "environment not found" proxy error (a 500) only
// counts as gone when ForgetMissingEnvironments is enabled, and the match is
// deliberately narrow: a transient proxy/connectivity 500 must NOT be treated as
// gone, otherwise a temporary outage would wipe resources from state on refresh.
func TestIsResourceGone(t *testing.T) {
	const (
		notFound404 = `arcane API error: 404 Not Found: {"detail":"project not found"}`
		envNotFound = `arcane API error: 500 Internal Server Error: {"detail":"failed to proxy request to environment: failed to get environment: environment not found"}`
		// Transient failures also surface as 500s from the proxy, but with
		// different text. These must never be classified as gone.
		transientRefused = `arcane API error: 500 Internal Server Error: {"detail":"failed to proxy request to environment: dial tcp 10.0.0.5:3552: connect: connection refused"}`
		transientTimeout = `arcane API error: 500 Internal Server Error: {"detail":"failed to proxy request to environment: context deadline exceeded"}`
	)

	cases := []struct {
		name   string
		forget bool
		err    error
		want   bool
	}{
		{"nil error, flag off", false, nil, false},
		{"nil error, flag on", true, nil, false},

		{"404 is always gone, flag off", false, errors.New(notFound404), true},
		{"404 is always gone, flag on", true, errors.New(notFound404), true},

		{"env not found stays an error when flag off", false, errors.New(envNotFound), false},
		{"env not found is gone when flag on", true, errors.New(envNotFound), true},
		{"env not found match is case-insensitive", true, errors.New("FAILED: Environment Not Found"), true},

		{"transient connection refused is not gone, flag on", true, errors.New(transientRefused), false},
		{"transient timeout is not gone, flag on", true, errors.New(transientTimeout), false},
		{"unrelated error is not gone, flag on", true, errors.New("boom"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Client{ForgetMissingEnvironments: tc.forget}
			if got := c.IsResourceGone(tc.err); got != tc.want {
				t.Fatalf("IsResourceGone(%v) with ForgetMissingEnvironments=%v = %v, want %v", tc.err, tc.forget, got, tc.want)
			}
		})
	}
}
