package evaluator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveFileRejectsEscapes(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	root := filepath.Join(parent, "workspace")
	if err := os.MkdirAll(filepath.Join(root, ".cue-eval"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, ".cue-eval", "project.json")
	if err := os.WriteFile(manifest, []byte(`{"apiVersion":"cue-eval.dev/project/v1alpha1","packs":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "outside.json")
	if err := os.WriteFile(outside, []byte(`{"outside":true}`), 0o600); err != nil {
		t.Fatal(err)
	}

	w, err := OpenWorkspace(root, "", DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.ResolveFile(".cue-eval/project.json"); err != nil {
		t.Fatalf("resolve in-root file: %v", err)
	}

	for _, path := range []string{"", "../outside.json", outside} {
		path := path
		t.Run(path, func(t *testing.T) {
			if _, err := w.ResolveFile(path); err == nil {
				t.Fatalf("ResolveFile(%q) unexpectedly succeeded", path)
			}
		})
	}

	link := filepath.Join(root, "outside-link.json")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := w.ResolveFile("outside-link.json"); err == nil {
		t.Fatal("ResolveFile accepted a symlink escaping the workspace")
	}
}

func TestProjectManifestRejectsTrailingJSON(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".cue-eval"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := []byte(`{"apiVersion":"cue-eval.dev/project/v1alpha1","packs":[]} {}`)
	if err := os.WriteFile(filepath.Join(root, ".cue-eval", "project.json"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	w, err := OpenWorkspace(root, "", DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if _, err := New(w).Discover(context.Background(), DiscoverRequest{}); err == nil {
		t.Fatal("manifest with trailing JSON unexpectedly succeeded")
	}
}

func TestDefaultLimitsAreBounded(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()
	if limits.MaxBatchItems <= 0 || limits.MaxInputBytes <= 0 ||
		limits.MaxContractBytes <= 0 || limits.MaxContractFiles <= 0 ||
		limits.MaxDiagnostics <= 0 || limits.MaxDiagnosticBytes <= 0 {
		t.Fatalf("default limits must all be positive: %+v", limits)
	}
}

func TestWithinInputRootsUsesPathBoundaries(t *testing.T) {
	t.Parallel()

	roots := []string{"allowed/fixtures"}
	for _, path := range []string{"allowed/fixtures/item.json", "allowed/fixtures/nested/item.json"} {
		if !withinInputRoots(path, roots) {
			t.Fatalf("withinInputRoots(%q) = false", path)
		}
	}
	for _, path := range []string{"", "../item.json", "/absolute/item.json", "allowed/fixtures-other/item.json"} {
		if withinInputRoots(path, roots) {
			t.Fatalf("withinInputRoots(%q) = true", path)
		}
	}
}
