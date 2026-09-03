package sdkclient

import (
	"encoding/json"
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

// TestGitOpsSyncRequestPreDeployMarshaling locks in the wire format of the
// pre-deploy lifecycle hook fields: nil pointers stay out of the JSON entirely
// (the API keeps whatever is configured for an absent field), while a set
// pointer is encoded under the exact camelCase key the API expects — including
// an explicit empty string, which is how an existing hook is cleared.
func TestGitOpsSyncRequestPreDeployMarshaling(t *testing.T) {
	preDeployKeys := []string{
		"preDeployScriptPath", "preDeployRunnerImage", "preDeployEnv",
		"preDeployExtraMounts", "preDeployTimeoutSec", "preDeployNetworkMode",
	}

	marshal := func(v any) map[string]json.RawMessage {
		t.Helper()
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal %T: %s", v, err)
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatalf("unmarshal %T: %s", v, err)
		}
		return m
	}

	// Unset fields are omitted, on both request types.
	for _, m := range []map[string]json.RawMessage{
		marshal(GitOpsSyncCreateRequest{Name: "s"}),
		marshal(GitOpsSyncUpdateRequest{}),
	} {
		for _, k := range preDeployKeys {
			if v, ok := m[k]; ok {
				t.Errorf("unset %s was encoded as %s", k, v)
			}
		}
	}

	scriptPath := "pre-deploy.sh"
	runnerImage := "ghcr.io/getsops/sops:v3.11.0"
	env := "SOPS_AGE_KEY_FILE=/run/secrets/age.key"
	mounts := "/opt/arcane/secrets/age.key:/run/secrets/age.key:ro"
	timeout := int64(120)
	networkMode := "" // empty string clears / resets to the server default

	m := marshal(GitOpsSyncUpdateRequest{
		PreDeployScriptPath:  &scriptPath,
		PreDeployRunnerImage: &runnerImage,
		PreDeployEnv:         &env,
		PreDeployExtraMounts: &mounts,
		PreDeployTimeoutSec:  &timeout,
		PreDeployNetworkMode: &networkMode,
	})
	want := map[string]string{
		"preDeployScriptPath":  `"pre-deploy.sh"`,
		"preDeployRunnerImage": `"ghcr.io/getsops/sops:v3.11.0"`,
		"preDeployEnv":         `"SOPS_AGE_KEY_FILE=/run/secrets/age.key"`,
		"preDeployExtraMounts": `"/opt/arcane/secrets/age.key:/run/secrets/age.key:ro"`,
		"preDeployTimeoutSec":  `120`,
		"preDeployNetworkMode": `""`,
	}
	for k, w := range want {
		if got, ok := m[k]; !ok {
			t.Errorf("%s missing from encoded update request", k)
		} else if string(got) != w {
			t.Errorf("%s: got %s, want %s", k, got, w)
		}
	}
}
