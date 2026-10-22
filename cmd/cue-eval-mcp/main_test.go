package main

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/reductist/cue-eval-mcp/internal/evaluator"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestDiscoverCLI(t *testing.T) {
	var out, errout bytes.Buffer
	err := run(context.Background(), []string{"discover", "--workspace", repoRoot(t), "--query", "toy"}, bytes.NewReader(nil), &out, &errout)
	if err != nil {
		t.Fatalf("run: %v stderr=%s", err, errout.String())
	}
	var got evaluator.DiscoverResult
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Packs) != 1 || got.Packs[0].ID != "toy-plugin" {
		t.Fatalf("result=%+v", got)
	}
}

func TestEvaluateCLIParity(t *testing.T) {
	root := repoRoot(t)
	req := evaluator.EvaluateRequest{Items: []evaluator.EvaluateItem{{ID: "valid", Mode: "implementation", Pack: "toy-plugin", Target: "manifest", Format: "json", Source: evaluator.Source{Type: "file", Path: ".cue-eval/toy-plugin/fixtures/positive.json"}}}}
	b, _ := json.Marshal(req)
	var out, errout bytes.Buffer
	if err := run(context.Background(), []string{"evaluate", "--workspace", root}, bytes.NewReader(b), &out, &errout); err != nil {
		t.Fatalf("run: %v stderr=%s", err, errout.String())
	}
	var got evaluator.EvaluateResult
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Items[0].Outcome != "valid-concrete" || got.Items[0].GateEligible {
		t.Fatalf("item=%+v", got.Items[0])
	}
}

func TestEvaluateCLIFailsOnInvalid(t *testing.T) {
	root := repoRoot(t)
	req := evaluator.EvaluateRequest{Items: []evaluator.EvaluateItem{{ID: "invalid", Mode: "implementation", Pack: "toy-plugin", Target: "manifest", Format: "json", Source: evaluator.Source{Type: "file", Path: ".cue-eval/toy-plugin/fixtures/negative.json"}}}}
	b, _ := json.Marshal(req)
	var out, errout bytes.Buffer
	if err := run(context.Background(), []string{"evaluate", "--workspace", root}, bytes.NewReader(b), &out, &errout); err == nil {
		t.Fatal("invalid evaluation returned success")
	}
	var got evaluator.EvaluateResult
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Items[0].Outcome != "invalid-concrete" {
		t.Fatalf("item=%+v", got.Items[0])
	}
}

func TestEvaluateCLIFailsOnTopLevelRequestError(t *testing.T) {
	var out, errout bytes.Buffer
	err := run(context.Background(), []string{"evaluate", "--workspace", repoRoot(t)}, bytes.NewReader([]byte(`{"items":[]}`)), &out, &errout)
	if err == nil {
		t.Fatal("empty batch returned success")
	}
	var got evaluator.EvaluateResult
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Diagnostics) != 1 || got.Diagnostics[0].Code != "batch-limit" {
		t.Fatalf("diagnostics=%+v", got.Diagnostics)
	}
}

func TestEvaluateCLIRejectsTrailingJSON(t *testing.T) {
	var out, errout bytes.Buffer
	err := run(context.Background(), []string{"evaluate", "--workspace", repoRoot(t)}, bytes.NewReader([]byte(`{"items":[]} {}`)), &out, &errout)
	if err == nil {
		t.Fatal("trailing JSON returned success")
	}
}

func TestUnknownCommandDoesNotRequireWorkspace(t *testing.T) {
	var out, errout bytes.Buffer
	err := run(context.Background(), []string{"unknown", "--workspace", "/does/not/exist"}, bytes.NewReader(nil), &out, &errout)
	if err == nil || err.Error() != `unknown command "unknown"` {
		t.Fatalf("err=%v", err)
	}
}
