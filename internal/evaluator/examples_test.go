package evaluator

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
)

func repositoryEngine(t *testing.T) *Engine {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	w, err := OpenWorkspace(root, "", DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return New(w)
}

func TestRepositoryExamples(t *testing.T) {
	t.Parallel()
	e := repositoryEngine(t)

	tests := []struct {
		name, mode, pack, target, format, expression, path, outcome string
	}{
		{"toy positive", "implementation", "toy-plugin", "manifest", "json", "", ".cue-eval/toy-plugin/fixtures/positive.json", "valid-concrete"},
		{"toy negative", "implementation", "toy-plugin", "manifest", "json", "", ".cue-eval/toy-plugin/fixtures/negative.json", "invalid-concrete"},
		{"toy incomplete", "author", "toy-plugin", "manifest", "cue", "#CandidateManifest", ".cue-eval/toy-plugin/fixtures/incomplete.cue", "compatible-so-far"},
		{"toy conflict", "author", "toy-plugin", "manifest", "cue", "#CandidateManifest", ".cue-eval/toy-plugin/fixtures/conflicting.cue", "conflict"},
		{"puppet valid", "implementation", "puppet-domain-example", "effective-node-data", "json", "", "integrations/puppet/examples/fixtures/valid.json", "valid-concrete"},
		{"puppet invalid combination", "implementation", "puppet-domain-example", "effective-node-data", "json", "", "integrations/puppet/examples/fixtures/invalid-combination.json", "invalid-concrete"},
		{"puppet hostname mismatch", "implementation", "puppet-domain-example", "effective-node-data", "json", "", "integrations/puppet/examples/fixtures/hostname-mismatch.json", "invalid-concrete"},
		{"puppet incomplete effective", "implementation", "puppet-domain-example", "effective-node-data", "json", "", "integrations/puppet/examples/fixtures/incomplete.json", "invalid-concrete"},
		{"puppet source fragment", "implementation", "puppet-domain-example", "source-fragment", "json", "", "integrations/puppet/examples/fixtures/incomplete.json", "valid-concrete"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			r := e.Evaluate(context.Background(), EvaluateRequest{Items: []EvaluateItem{{
				ID: tc.name, Mode: tc.mode, Pack: tc.pack, Target: tc.target,
				Format: tc.format, Expression: tc.expression,
				Source: Source{Type: "file", Path: tc.path},
			}}})
			if len(r.Items) != 1 || r.Items[0].Outcome != tc.outcome {
				t.Fatalf("outcome = %#v, want %q", r.Items, tc.outcome)
			}
			if r.Items[0].GateEligible {
				t.Fatal("declared review metadata must not make an example gate eligible")
			}
		})
	}
}

func TestDiscoveryIsDeterministic(t *testing.T) {
	t.Parallel()
	e := repositoryEngine(t)
	r, err := e.Discover(context.Background(), DiscoverRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Packs) != 2 || r.Packs[0].ID != "puppet-domain-example" || r.Packs[1].ID != "toy-plugin" {
		t.Fatalf("unexpected discovery order: %+v", r.Packs)
	}
	for _, pack := range r.Packs {
		if pack.ContractDigest == "" || pack.ReviewVerified {
			t.Fatalf("unexpected discovery provenance: %+v", pack)
		}
	}
}
