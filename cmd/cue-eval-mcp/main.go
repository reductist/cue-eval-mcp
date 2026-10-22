package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/reductist/cue-eval-mcp/internal/evaluator"
	"github.com/reductist/cue-eval-mcp/internal/mcpserver"
)

var version = evaluator.KitVersion

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	const usage = "usage: cue-eval-mcp <serve|discover|inspect|evaluate|version> [options]"
	if len(args) == 0 {
		return errors.New(usage)
	}
	if args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		_, err := fmt.Fprintln(stdout, usage)
		return err
	}
	if args[0] == "version" {
		_, err := fmt.Fprintln(stdout, version)
		return err
	}
	if args[0] != "serve" && args[0] != "discover" && args[0] != "inspect" && args[0] != "evaluate" {
		return fmt.Errorf("unknown command %q", args[0])
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)
	workspace := fs.String("workspace", ".", "workspace root")
	manifest := fs.String("manifest", ".cue-eval/project.json", "workspace-relative project manifest")
	var query, pack, target, request *string
	var limit *int
	var source *bool
	switch args[0] {
	case "discover":
		query = fs.String("query", "", "case-insensitive metadata query")
		limit = fs.Int("limit", 100, "maximum packs")
	case "inspect":
		pack = fs.String("pack", "", "pack id")
		target = fs.String("target", "", "target name or definition")
		source = fs.Bool("include-source", false, "include exact contract source")
	case "evaluate":
		request = fs.String("request", "-", "JSON request file or - for stdin")
	}
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	w, err := evaluator.OpenWorkspace(*workspace, *manifest, evaluator.DefaultLimits())
	if err != nil {
		return err
	}
	defer w.Close()
	engine := evaluator.New(w)
	switch args[0] {
	case "serve":
		if fs.NArg() != 0 {
			return errors.New("serve accepts only options")
		}
		mcpserver.Version = version
		logger := slog.New(slog.NewTextHandler(stderr, nil))
		return mcpserver.RunStdio(ctx, engine, logger)
	case "discover":
		out, err := engine.Discover(ctx, evaluator.DiscoverRequest{Query: *query, Limit: *limit})
		if err != nil {
			return err
		}
		return writeJSON(stdout, out)
	case "inspect":
		if *pack == "" {
			return errors.New("inspect requires --pack")
		}
		out, err := engine.Inspect(ctx, evaluator.InspectRequest{Pack: *pack, Target: *target, IncludeSource: *source})
		if err != nil {
			return err
		}
		return writeJSON(stdout, out)
	case "evaluate":
		var reader io.Reader = stdin
		if *request != "-" {
			f, err := os.Open(*request)
			if err != nil {
				return err
			}
			defer f.Close()
			reader = f
		}
		const maxRequestBytes = 8 << 20
		raw, err := io.ReadAll(io.LimitReader(reader, maxRequestBytes+1))
		if err != nil {
			return err
		}
		if len(raw) > maxRequestBytes {
			return errors.New("request exceeds 8 MiB limit")
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		var req evaluator.EvaluateRequest
		if err := dec.Decode(&req); err != nil {
			return fmt.Errorf("decode request: %w", err)
		}
		var extra any
		if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
			if err == nil {
				return errors.New("request contains trailing JSON")
			}
			return fmt.Errorf("decode trailing request data: %w", err)
		}
		out := engine.Evaluate(ctx, req)
		if err := writeJSON(stdout, out); err != nil {
			return err
		}
		for _, d := range out.Diagnostics {
			if d.Severity == "error" {
				return exitFailure{}
			}
		}
		for _, item := range out.Items {
			if item.Outcome == "error" || item.Outcome == "invalid-concrete" || item.Outcome == "conflict" {
				return exitFailure{}
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

type exitFailure struct{}

func (exitFailure) Error() string { return "evaluation did not pass" }

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
