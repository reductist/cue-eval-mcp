package evaluator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/parser"
)

func transportSchema(t *testing.T) (cue.Value, *cue.Context) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	ctx := cuecontext.New()
	buildContext := build.NewContext()
	instance := buildContext.NewInstance(filepath.Join(root, ".cue-eval", "api"), nil)
	for _, name := range []string{"request.cue", "result.cue"} {
		contents, err := readTestFile(filepath.Join(root, ".cue-eval", "api", name))
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(name, contents, parser.AllErrors)
		if err != nil {
			t.Fatal(err)
		}
		if err := instance.AddSyntax(file); err != nil {
			t.Fatal(err)
		}
	}
	if err := instance.Complete(); err != nil {
		t.Fatal(err)
	}
	value := ctx.BuildInstance(instance)
	if err := value.Err(); err != nil {
		t.Fatal(err)
	}
	return value, ctx
}

func readTestFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func assertWireValue(t *testing.T, schema cue.Value, ctx *cue.Context, definition string, value any) {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	data := ctx.CompileBytes(b, cue.Filename("wire.json"))
	if err := data.Err(); err != nil {
		t.Fatal(err)
	}
	target := schema.LookupPath(cue.ParsePath(definition))
	if err := target.Unify(data).Validate(cue.Final(), cue.Concrete(true)); err != nil {
		t.Fatalf("%s rejected runtime JSON %s: %v", definition, b, err)
	}
}

func assertWireRejected(t *testing.T, schema cue.Value, ctx *cue.Context, definition string, value any) {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	data := ctx.CompileBytes(b, cue.Filename("wire.json"))
	if err := schema.LookupPath(cue.ParsePath(definition)).Unify(data).Validate(cue.Final(), cue.Concrete(true)); err == nil {
		t.Fatalf("%s unexpectedly accepted runtime JSON %s", definition, b)
	}
}

func TestRuntimeJSONConformsToCUESchemas(t *testing.T) {
	t.Parallel()
	schema, ctx := transportSchema(t)
	e := repositoryEngine(t)

	req := EvaluateRequest{APIVersion: APIVersion, Items: []EvaluateItem{{
		ID: "valid", Mode: "implementation", Pack: "toy-plugin", Target: "manifest", Format: "json",
		Source: Source{Type: "file", Path: ".cue-eval/toy-plugin/fixtures/positive.json"},
	}}}
	assertWireValue(t, schema, ctx, "#EvaluateRequest", req)
	assertWireRejected(t, schema, ctx, "#EvaluateRequest", EvaluateRequest{})
	assertWireValue(t, schema, ctx, "#EvaluateResult", e.Evaluate(context.Background(), req))

	author := EvaluateRequest{Items: []EvaluateItem{{
		ID: "partial", Mode: "author", Pack: "toy-plugin", Target: "manifest", Format: "cue", Expression: "#CandidateManifest",
		Source: Source{Type: "file", Path: ".cue-eval/toy-plugin/fixtures/incomplete.cue"},
	}}}
	assertWireValue(t, schema, ctx, "#EvaluateRequest", author)
	assertWireValue(t, schema, ctx, "#EvaluateResult", e.Evaluate(context.Background(), author))
	assertWireRejected(t, schema, ctx, "#EvaluateRequest", EvaluateRequest{Items: []EvaluateItem{{
		ID: "cue-implementation", Mode: "implementation", Pack: "toy-plugin", Target: "manifest", Format: "cue", Expression: "#CandidateManifest",
		Source: Source{Type: "file", Path: ".cue-eval/toy-plugin/fixtures/incomplete.cue"},
	}}})

	assertWireValue(t, schema, ctx, "#EvaluateResult", e.Evaluate(context.Background(), EvaluateRequest{}))
	assertWireValue(t, schema, ctx, "#EvaluateResult", e.Evaluate(context.Background(), EvaluateRequest{Items: []EvaluateItem{{ID: "bad", Mode: "unsupported"}}}))
}
