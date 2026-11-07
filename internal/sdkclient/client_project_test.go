package sdkclient

import (
	"context"
	"encoding/json"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCreateProject_MultipartWireContract pins the project create contract:
// Arcane takes it as multipart/form-data with a JSON "project" part and a JSON
// workspace "manifest" part, both required. Sending a plain JSON body is
// rejected with "cannot read multipart form: request Content-Type isn't
// multipart/form-data", so a regression here breaks every project resource.
func TestCreateProject_MultipartWireContract(t *testing.T) {
	var gotPath, gotMethod, gotContentType string
	parts := map[string]map[string]any{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		gotContentType = r.Header.Get("Content-Type")

		mediaType, params, err := mime.ParseMediaType(gotContentType)
		if err != nil || mediaType != "multipart/form-data" {
			t.Errorf("content type: got %q, want multipart/form-data", gotContentType)
			w.Write([]byte(`{"success":true,"data":{}}`))
			return
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		form, err := mr.ReadForm(1 << 20)
		if err != nil {
			t.Fatalf("reading multipart form: %v", err)
		}
		for name, values := range form.Value {
			var decoded map[string]any
			if err := json.Unmarshal([]byte(values[0]), &decoded); err != nil {
				t.Errorf("part %q is not JSON: %v", name, err)
				continue
			}
			parts[name] = decoded
		}
		w.Write([]byte(`{"success":true,"data":{"id":"proj-1","name":"web"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k")
	env := "FOO=bar"
	out, err := c.CreateProject(context.Background(), "env-1", ProjectCreateRequest{
		Name:           "web",
		ComposeContent: "services: {}",
		EnvContent:     &env,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if out.ID != "proj-1" {
		t.Errorf("decoded id: got %q, want proj-1", out.ID)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method: got %s, want POST", gotMethod)
	}
	if gotPath != "/environments/env-1/projects" {
		t.Errorf("path: got %q", gotPath)
	}

	project, ok := parts["project"]
	if !ok {
		t.Fatalf("request is missing the %q part; got parts %v", "project", keys(anyMap(parts)))
	}
	for key, want := range map[string]any{"name": "web", "composeContent": "services: {}", "envContent": "FOO=bar"} {
		if project[key] != want {
			t.Errorf("project part %q: got %v, want %v", key, project[key], want)
		}
	}

	manifest, ok := parts["manifest"]
	if !ok {
		t.Fatalf("request is missing the %q part; got parts %v", "manifest", keys(anyMap(parts)))
	}
	changes, ok := manifest["fileChanges"].([]any)
	if !ok {
		t.Fatalf("manifest fileChanges: got %#v, want a JSON array", manifest["fileChanges"])
	}
	if len(changes) != 0 {
		t.Errorf("manifest fileChanges: got %v, want empty", changes)
	}
}

func anyMap[V any](m map[string]V) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// TestProjectOperationStream pins how up, redeploy and pull read Arcane's
// answer: HTTP 200 and a stream of JSON lines, where a failure is an
// {"error": ...} line and success ends with {"done": true}. A 200 alone must
// not count as success, or a failed deploy applies cleanly.
func TestProjectOperationStream(t *testing.T) {
	cases := []struct {
		name    string
		ctype   string
		body    string
		wantErr string
	}{
		{"done", "application/x-json-stream", `{"type":"activity","activityId":"a1"}` + "\n" + `{"status":"Pulling"}` + "\nraw docker output\n" + `{"done":true}` + "\n", ""},
		{"error line", "application/x-json-stream", `{"type":"activity","activityId":"a1"}` + "\n" + `{"error":"pull access denied for web"}` + "\n", "pull access denied for web"},
		{"docker error detail", "application/x-json-stream", `{"errorDetail":{"message":"manifest unknown"}}` + "\n", "manifest unknown"},
		{"cut short", "application/x-json-stream", `{"type":"activity","activityId":"a1"}` + "\n", "ended before the operation finished"},
		{"not a stream", "application/json", `{"success":true}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tc.ctype)
				w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			c := NewClient(srv.URL, "k")
			ops := map[string]func() error{
				"up":       func() error { return c.UpProject(context.Background(), "0", "p", nil) },
				"deploy":   func() error { return c.DeployProject(context.Background(), "0", "p") },
				"redeploy": func() error { return c.RedeployProject(context.Background(), "0", "p") },
				"pull":     func() error { return c.PullProjectImages(context.Background(), "0", "p") },
			}
			for op, run := range ops {
				err := run()
				switch {
				case tc.wantErr == "" && err != nil:
					t.Errorf("%s: unexpected error: %v", op, err)
				case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
					t.Errorf("%s: got %v, want an error containing %q", op, err, tc.wantErr)
				}
			}
		})
	}
}
