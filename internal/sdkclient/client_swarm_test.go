package sdkclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDeploySwarmStack_WireContract pins the v2.0.1 deploy contract: the POST
// path, the camelCase request field names, and decoding of the {success,data}
// envelope. A casing or path regression would break against a real server.
func TestDeploySwarmStack_WireContract(t *testing.T) {
	var gotPath, gotMethod string
	var raw map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &raw)
		w.Write([]byte(`{"success":true,"data":{"name":"web"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k")
	env := "release"
	out, err := c.DeploySwarmStack(context.Background(), "env-1", SwarmStackDeployRequest{
		Name:           "web",
		ComposeContent: "services: {}",
		EnvContent:     &env,
	})
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if out.Name != "web" {
		t.Errorf("decoded name: got %q, want web", out.Name)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method: got %s, want POST", gotMethod)
	}
	if gotPath != "/environments/env-1/swarm/stacks" {
		t.Errorf("path: got %q", gotPath)
	}
	for _, key := range []string{"name", "composeContent", "envContent"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("request body missing camelCase field %q; got keys %v", key, keys(raw))
		}
	}
}

// TestEncodeSwarmSecretData verifies the base64 encoding used for secret data.
func TestEncodeSwarmSecretData(t *testing.T) {
	if got := EncodeSwarmSecretData("hello"); got != "aGVsbG8=" {
		t.Errorf("EncodeSwarmSecretData(hello) = %q, want aGVsbG8=", got)
	}
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
