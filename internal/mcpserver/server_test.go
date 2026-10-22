package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"slices"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/reductist/cue-eval-mcp/internal/evaluator"
)

type fakeBackend struct{ discoverErr error }

func (f *fakeBackend) Discover(_ context.Context, _ evaluator.DiscoverRequest) (evaluator.DiscoverResult, error) {
	if f.discoverErr != nil {
		return evaluator.DiscoverResult{}, f.discoverErr
	}
	return evaluator.DiscoverResult{APIVersion: evaluator.APIVersion, Packs: []evaluator.PackSummary{}}, nil
}
func (*fakeBackend) Inspect(_ context.Context, _ evaluator.InspectRequest) (evaluator.InspectResult, error) {
	return evaluator.InspectResult{APIVersion: evaluator.APIVersion}, nil
}
func (*fakeBackend) Evaluate(_ context.Context, _ evaluator.EvaluateRequest) evaluator.EvaluateResult {
	return evaluator.EvaluateResult{APIVersion: evaluator.APIVersion, Items: []evaluator.ItemResult{}}
}

func TestToolsOverStdio(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=TestStdioHelperProcess")
	command.Env = append(os.Environ(), "GO_WANT_CUE_EVAL_MCP_HELPER=1")
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	command.Stderr = io.Discard
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	encoder, decoder := json.NewEncoder(stdin), json.NewDecoder(stdout)
	writeRPC(t, encoder, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{"protocolVersion": "2025-06-18", "capabilities": map[string]any{}, "clientInfo": map[string]any{"name": "transport-test", "version": "1"}}})
	initialize := readRPC(t, decoder, 1)
	var initialized struct {
		ProtocolVersion string `json:"protocolVersion"`
		ServerInfo      struct {
			Name string `json:"name"`
		} `json:"serverInfo"`
	}
	if err := json.Unmarshal(initialize.Result, &initialized); err != nil {
		t.Fatal(err)
	}
	if initialized.ProtocolVersion != "2025-06-18" || initialized.ServerInfo.Name != Name {
		t.Fatalf("initialize=%#v", initialized)
	}
	writeRPC(t, encoder, map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
	writeRPC(t, encoder, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": map[string]any{}})
	listedResult := readRPC(t, decoder, 2)
	var listed struct {
		Tools []struct {
			Name        string          `json:"name"`
			InputSchema json.RawMessage `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(listedResult.Result, &listed); err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range listed.Tools {
		names = append(names, tool.Name)
		if len(tool.InputSchema) == 0 {
			t.Errorf("tool %q has no input schema", tool.Name)
		}
	}
	slices.Sort(names)
	want := []string{"discover_contracts", "evaluate", "inspect_contract"}
	if !slices.Equal(names, want) {
		t.Fatalf("tools=%v want %v", names, want)
	}
	writeRPC(t, encoder, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "tools/call", "params": map[string]any{"name": "discover_contracts", "arguments": map[string]any{"query": "toy", "limit": 2}}})
	callResult := readRPC(t, decoder, 3)
	var called struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(callResult.Result, &called); err != nil {
		t.Fatal(err)
	}
	if called.IsError || len(called.Content) != 1 || called.Content[0].Type != "text" {
		t.Fatalf("call=%#v", called)
	}
	writeRPC(t, encoder, map[string]any{"jsonrpc": "2.0", "id": 4, "method": "tools/call", "params": map[string]any{"name": "inspect_contract", "arguments": map[string]any{"pack": "toy-plugin", "target": "manifest"}}})
	assertSuccessfulToolCall(t, readRPC(t, decoder, 4))
	writeRPC(t, encoder, map[string]any{"jsonrpc": "2.0", "id": 5, "method": "tools/call", "params": map[string]any{"name": "evaluate", "arguments": map[string]any{"items": []any{map[string]any{
		"id": "inline", "mode": "implementation", "pack": "toy-plugin", "target": "manifest", "format": "json",
		"source": map[string]any{"type": "inline", "content": `{}`},
	}}}}})
	assertSuccessfulToolCall(t, readRPC(t, decoder, 5))
	if err := stdin.Close(); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("server exit: %v", err)
	}
}

func assertSuccessfulToolCall(t *testing.T, response rpcResponse) {
	t.Helper()
	var called struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(response.Result, &called); err != nil {
		t.Fatal(err)
	}
	if called.IsError || len(called.Content) != 1 || called.Content[0].Type != "text" {
		t.Fatalf("call=%#v", called)
	}
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   json.RawMessage `json:"error"`
}

func writeRPC(t *testing.T, e *json.Encoder, v any) {
	t.Helper()
	if err := e.Encode(v); err != nil {
		t.Fatal(err)
	}
}
func readRPC(t *testing.T, d *json.Decoder, id int) rpcResponse {
	t.Helper()
	var r rpcResponse
	if err := d.Decode(&r); err != nil {
		t.Fatal(err)
	}
	if r.JSONRPC != "2.0" || r.ID != id || len(r.Error) != 0 {
		t.Fatalf("response=%#v", r)
	}
	return r
}
func TestStdioHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_CUE_EVAL_MCP_HELPER") != "1" {
		return
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := RunStdio(context.Background(), &fakeBackend{}, logger); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func TestToolErrorsAreVisibleToClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	st, ct := mcp.NewInMemoryTransports()
	server := New(&fakeBackend{discoverErr: errors.New("bounded failure")}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx, st) }()
	client := mcp.NewClient(&mcp.Implementation{Name: "transport-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "discover_contracts", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("backend error was not returned as a tool error")
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("server did not stop")
	}
}
func TestNewRejectsNilBackend(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("New(nil) did not panic")
		}
	}()
	New(nil, nil)
}
