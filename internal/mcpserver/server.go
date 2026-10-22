// Package mcpserver exposes the evaluator as a local, read-only MCP server.
package mcpserver

import (
	"context"
	"log/slog"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/reductist/cue-eval-mcp/internal/evaluator"
)

const Name = "cue-eval-mcp"

var Version = evaluator.KitVersion

type Engine interface {
	Discover(context.Context, evaluator.DiscoverRequest) (evaluator.DiscoverResult, error)
	Inspect(context.Context, evaluator.InspectRequest) (evaluator.InspectResult, error)
	Evaluate(context.Context, evaluator.EvaluateRequest) evaluator.EvaluateResult
}

func New(engine Engine, logger *slog.Logger) *mcp.Server {
	if engine == nil {
		panic("mcpserver: nil engine")
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	server := mcp.NewServer(&mcp.Implementation{Name: Name, Version: Version}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{},
		Instructions: "Discover and inspect applicable contracts before evaluation. Author-mode compatibility is incomplete and never gate-eligible.",
		Logger:       logger,
	})
	mcp.AddTool(server, &mcp.Tool{Name: "discover_contracts", Description: "Return bounded metadata for configured contract packs. Read-only; never initializes a workspace."}, func(ctx context.Context, _ *mcp.CallToolRequest, in evaluator.DiscoverRequest) (*mcp.CallToolResult, any, error) {
		out, err := engine.Discover(ctx, in)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "inspect_contract", Description: "Return bounded metadata and optionally source for one configured contract target. Read-only."}, func(ctx context.Context, _ *mcp.CallToolRequest, in evaluator.InspectRequest) (*mcp.CallToolResult, any, error) {
		out, err := engine.Inspect(ctx, in)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "evaluate", Description: "Evaluate a bounded batch in author or implementation mode and return per-item structured outcomes."}, func(ctx context.Context, _ *mcp.CallToolRequest, in evaluator.EvaluateRequest) (*mcp.CallToolResult, any, error) {
		return nil, engine.Evaluate(ctx, in), nil
	})
	return server
}

func RunStdio(ctx context.Context, engine Engine, logger *slog.Logger) error {
	return New(engine, logger).Run(ctx, &mcp.StdioTransport{MaxLineLength: 2 << 20})
}
