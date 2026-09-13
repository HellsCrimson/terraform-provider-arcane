package sdkclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCreateContainerRegistry_AlwaysSendsRepositoryNames is a regression test
// for arcane rejecting a create body that omits the property entirely with
// "expected required property repositoryNames to be present". The server
// requires the key to exist even when there is nothing to configure, so a nil
// slice must still marshal as an empty array rather than being dropped or sent
// as null.
func TestCreateContainerRegistry_AlwaysSendsRepositoryNames(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []string
		want []any
	}{
		{"unset", nil, []any{}},
		{"configured", []string{"team", "team/platform"}, []any{"team", "team/platform"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var raw map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(body, &raw)
				w.Write([]byte(`{"success":true,"data":{"id":"reg-1","repositoryNames":["team"]}}`))
			}))
			defer srv.Close()

			c := NewClient(srv.URL, "k")
			reg, err := c.CreateContainerRegistry(context.Background(), CreateContainerRegistryRequest{
				URL:             "https://ghcr.io",
				Username:        "bot",
				Token:           "t",
				RegistryType:    "generic",
				RepositoryNames: tc.in,
			})
			if err != nil {
				t.Fatalf("create: %v", err)
			}
			if len(reg.RepositoryNames) != 1 || reg.RepositoryNames[0] != "team" {
				t.Errorf("decoded repositoryNames: got %v, want [team]", reg.RepositoryNames)
			}

			got, ok := raw["repositoryNames"]
			if !ok {
				t.Fatalf("request body missing repositoryNames; got keys %v", keys(raw))
			}
			arr, ok := got.([]any)
			if !ok {
				t.Fatalf("repositoryNames: got %#v, want a JSON array", got)
			}
			if len(arr) != len(tc.want) {
				t.Fatalf("repositoryNames: got %v, want %v", arr, tc.want)
			}
			for i := range arr {
				if arr[i] != tc.want[i] {
					t.Errorf("repositoryNames[%d]: got %v, want %v", i, arr[i], tc.want[i])
				}
			}
		})
	}
}
