package evaluator

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testEngine(t *testing.T) *Engine {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	w, err := OpenWorkspace(root, "testdata/evaluator/project.json", DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	return New(w)
}

func TestImplementationSemantics(t *testing.T) {
	t.Parallel()
	e := testEngine(t)

	tests := []struct {
		name    string
		path    string
		outcome string
	}{
		{name: "default cannot fill evidence", path: "testdata/evaluator/basic/subjects/valid-default.json", outcome: "invalid-concrete"},
		{name: "nested required missing", path: "testdata/evaluator/basic/subjects/missing-nested-required.json", outcome: "invalid-concrete"},
		{name: "unexpected closed field", path: "testdata/evaluator/basic/subjects/unexpected-field.json", outcome: "invalid-concrete"},
		{name: "malformed", path: "testdata/evaluator/basic/subjects/malformed.json.invalid", outcome: "error"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			r := e.Evaluate(context.Background(), EvaluateRequest{Items: []EvaluateItem{{
				ID: "case", Mode: "implementation", Pack: "evaluator-basic", Target: "record", Format: "json",
				Source: Source{Type: "file", Path: tc.path},
			}}})
			if len(r.Items) != 1 || r.Items[0].Outcome != tc.outcome {
				t.Fatalf("outcome = %#v, want %q", r.Items, tc.outcome)
			}
			if r.Items[0].GateEligible {
				t.Fatal("candidate pack must never be gate eligible")
			}
		})
	}
}

func TestMissingDefaultIsRejectedAsMissingEvidence(t *testing.T) {
	t.Parallel()
	e := testEngine(t)
	r := e.Evaluate(context.Background(), EvaluateRequest{Items: []EvaluateItem{{
		ID: "default", Mode: "implementation", Pack: "evaluator-basic", Target: "record", Format: "json",
		Source: Source{Type: "file", Path: "testdata/evaluator/basic/subjects/valid-default.json"},
	}}})
	if got := r.Items[0].Outcome; got != "invalid-concrete" {
		t.Fatalf("outcome = %q", got)
	}
	if !hasDiagnostic(r.Items[0], "missing-evidence", "policy") {
		t.Fatalf("diagnostics = %+v, want missing policy evidence", r.Items[0].Diagnostics)
	}
}

func TestStrictEvidencePresence(t *testing.T) {
	t.Parallel()
	e := testEngine(t)

	completeJSON := `{
  "id":"record-one",
  "apiVersion":"example.test/v1",
  "derivedID":"record-one",
  "policy":"standard",
  "nullable":null,
  "nested":{"mode":"safe","items":[{"name":"web","port":80}]},
  "large":9007199254740993,
  "precise":0.1000000000000000001,
  "a.b":"literal",
  "a":{"b":"nested"}
}`
	completeYAML := `id: record-one
apiVersion: example.test/v1
derivedID: record-one
policy: standard
nullable: null
nested:
  mode: safe
  items:
    - name: web
      port: 80
large: 9007199254740993
precise: 0.1000000000000000001
a.b: literal
a:
  b: nested
`

	tests := []struct {
		name, format, content, outcome, missing string
	}{
		{name: "complete JSON preserves exact numbers and explicit null", format: "json", content: completeJSON, outcome: "valid-concrete"},
		{name: "complete YAML preserves nested lists and explicit null", format: "yaml", content: completeYAML, outcome: "valid-concrete"},
		{name: "YAML missing list element default", format: "yaml", content: strings.Replace(completeYAML, "      port: 80\n", "", 1), outcome: "invalid-concrete", missing: "nested.items[0].port"},
		{name: "optional absence is allowed", format: "json", content: completeJSON, outcome: "valid-concrete"},
		{name: "missing constant", format: "json", content: strings.Replace(completeJSON, "  \"apiVersion\":\"example.test/v1\",\n", "", 1), outcome: "invalid-concrete", missing: "apiVersion"},
		{name: "missing derived value", format: "json", content: strings.Replace(completeJSON, "  \"derivedID\":\"record-one\",\n", "", 1), outcome: "invalid-concrete", missing: "derivedID"},
		{name: "missing default", format: "json", content: strings.Replace(completeJSON, "  \"policy\":\"standard\",\n", "", 1), outcome: "invalid-concrete", missing: "policy"},
		{name: "missing nested constant", format: "json", content: strings.Replace(completeJSON, `"mode":"safe",`, "", 1), outcome: "invalid-concrete", missing: "nested.mode"},
		{name: "missing list element default", format: "json", content: strings.Replace(completeJSON, `,"port":80`, "", 1), outcome: "invalid-concrete", missing: "nested.items[0].port"},
		{name: "missing selector-sensitive label", format: "json", content: strings.Replace(completeJSON, "  \"a.b\":\"literal\",\n", "", 1), outcome: "invalid-concrete", missing: `"a.b"`},
		{name: "required nullable is not optional", format: "json", content: strings.Replace(completeJSON, "  \"nullable\":null,\n", "", 1), outcome: "invalid-concrete"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := e.Evaluate(context.Background(), EvaluateRequest{Items: []EvaluateItem{{
				ID: "strict", Mode: "implementation", Pack: "evaluator-basic", Target: "strict-evidence", Format: tc.format,
				Source: Source{Type: "inline", Content: tc.content},
			}}})
			if got := r.Items[0].Outcome; got != tc.outcome {
				t.Fatalf("outcome = %q, want %q; diagnostics=%+v", got, tc.outcome, r.Items[0].Diagnostics)
			}
			if tc.missing != "" && !hasDiagnostic(r.Items[0], "missing-evidence", tc.missing) {
				t.Fatalf("diagnostics = %+v, want missing path %q", r.Items[0].Diagnostics, tc.missing)
			}
		})
	}
}

func TestImplementationCUESourceIsRejected(t *testing.T) {
	t.Parallel()
	e := testEngine(t)
	r := e.Evaluate(context.Background(), EvaluateRequest{Items: []EvaluateItem{{
		ID: "cue", Mode: "implementation", Pack: "evaluator-basic", Target: "record", Format: "cue", Expression: "#Candidate",
		Source: Source{Type: "file", Path: "testdata/evaluator/basic/subjects/compatible.cue"},
	}}})
	if got := r.Items[0].Outcome; got != "error" {
		t.Fatalf("outcome = %q, want error", got)
	}
	if !hasDiagnostic(r.Items[0], "unsupported-implementation-format", "") {
		t.Fatalf("diagnostics = %+v", r.Items[0].Diagnostics)
	}
}

func hasDiagnostic(r ItemResult, code, path string) bool {
	for _, d := range r.Diagnostics {
		if d.Code == code && (path == "" || d.Path == path) {
			return true
		}
	}
	return false
}

func TestAuthorSemantics(t *testing.T) {
	t.Parallel()
	e := testEngine(t)
	for _, tc := range []struct {
		path, outcome string
	}{
		{"testdata/evaluator/basic/subjects/compatible.cue", "compatible-so-far"},
		{"testdata/evaluator/basic/subjects/conflict.cue", "conflict"},
	} {
		r := e.Evaluate(context.Background(), EvaluateRequest{Items: []EvaluateItem{{
			ID: "author", Mode: "author", Pack: "evaluator-basic", Target: "record", Format: "cue", Expression: "#Candidate",
			Source: Source{Type: "file", Path: tc.path},
		}}})
		if got := r.Items[0].Outcome; got != tc.outcome {
			t.Fatalf("%s outcome = %q, want %q; diagnostics=%v", tc.path, got, tc.outcome, r.Items[0].Diagnostics)
		}
		if r.Items[0].CompleteCheck || r.Items[0].GateEligible {
			t.Fatalf("author result must be incomplete and gate-ineligible: %+v", r.Items[0])
		}
	}
}

func TestDiagnosticsDoNotEchoSubjectLiterals(t *testing.T) {
	t.Parallel()
	e := testEngine(t)
	r := e.Evaluate(context.Background(), EvaluateRequest{Items: []EvaluateItem{{
		ID: "redaction", Mode: "implementation", Pack: "evaluator-basic", Target: "record", Format: "json",
		Source: Source{Type: "file", Path: "testdata/evaluator/basic/subjects/secret-mismatch.json"},
	}}})
	if got := r.Items[0].Outcome; got != "invalid-concrete" {
		t.Fatalf("outcome = %q", got)
	}
	for _, d := range r.Items[0].Diagnostics {
		if strings.Contains(d.Message, "SENSITIVE-VALUE-MUST-NOT-APPEAR") {
			t.Fatalf("diagnostic leaked subject literal: %+v", d)
		}
	}
}

func TestContractImportsAreRejected(t *testing.T) {
	t.Parallel()
	e := testEngine(t)
	_, err := e.Inspect(context.Background(), InspectRequest{Pack: "evaluator-imported"})
	if err == nil || !strings.Contains(err.Error(), "imports are disabled") {
		t.Fatalf("Inspect error = %v, want import rejection", err)
	}
}

func TestMalformedContractIsAnEvaluationError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".cue-eval", "malformed"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
  "apiVersion":"cue-eval.dev/project/v1alpha1",
  "packs":[{
    "id":"malformed-contract",
    "title":"Malformed contract fixture",
    "summary":"Exercises contract parse failures.",
    "root":".cue-eval/malformed",
    "review":{"state":"candidate"},
    "sources":["contracts.cue"],
    "inputRoots":[".cue-eval/malformed"],
    "targets":[{"name":"record","definition":"#Record","formats":["json"]}]
  }]
}`
	if err := os.WriteFile(filepath.Join(root, ".cue-eval", "project.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".cue-eval", "malformed", "contracts.cue"), []byte("package malformed\n#Record: {value:\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w, err := OpenWorkspace(root, "", DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = w.Close() })
	_, err = New(w).Inspect(context.Background(), InspectRequest{Pack: "malformed-contract"})
	if err == nil {
		t.Fatal("malformed contract unexpectedly inspected successfully")
	}
}

func TestBatchLimit(t *testing.T) {
	t.Parallel()
	e := testEngine(t)
	items := make([]EvaluateItem, DefaultLimits().MaxBatchItems+1)
	r := e.Evaluate(context.Background(), EvaluateRequest{Items: items})
	if len(r.Items) != 0 || len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != "batch-limit" {
		t.Fatalf("unexpected batch limit result: %+v", r)
	}
}
