package evaluator

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Workspace struct {
	Root, ManifestPath string
	Limits             Limits
	root               *os.Root
}

func OpenWorkspace(root, manifest string, limits Limits) (*Workspace, error) {
	if limits == (Limits{}) {
		limits = DefaultLimits()
	}
	if limits.MaxBatchItems <= 0 || limits.MaxInputBytes <= 0 || limits.MaxContractBytes <= 0 || limits.MaxContractFiles <= 0 || limits.MaxDiagnostics <= 0 || limits.MaxDiagnosticBytes <= 0 {
		return nil, errors.New("all evaluator limits must be positive")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace: %w", err)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace symlinks: %w", err)
	}
	info, err := os.Stat(real)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("workspace is not a directory: %s", real)
	}
	if manifest == "" {
		manifest = ".cue-eval/project.json"
	}
	rootHandle, err := os.OpenRoot(real)
	if err != nil {
		return nil, fmt.Errorf("open workspace root: %w", err)
	}
	w := &Workspace{Root: real, ManifestPath: manifest, Limits: limits, root: rootHandle}
	if _, err := w.ResolveFile(manifest); err != nil {
		_ = rootHandle.Close()
		return nil, fmt.Errorf("project manifest: %w", err)
	}
	return w, nil
}

func (w *Workspace) ResolveFile(name string) (string, error) {
	if name == "" || filepath.IsAbs(name) {
		return "", errors.New("path must be a non-empty workspace-relative path")
	}
	clean := filepath.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes workspace")
	}
	info, err := w.root.Stat(clean)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("path is not a regular file")
	}
	return filepath.Join(w.Root, clean), nil
}

func (w *Workspace) readFile(name string, limit int64) ([]byte, error) {
	path, err := w.ResolveFile(name)
	if err != nil {
		return nil, err
	}
	rel := w.relative(path)
	f, err := w.root.Open(filepath.FromSlash(rel))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errors.New("byte limit exceeded")
	}
	return b, nil
}

func (w *Workspace) Close() error {
	if w.root == nil {
		return nil
	}
	return w.root.Close()
}

func (w *Workspace) relative(path string) string {
	rel, err := filepath.Rel(w.Root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}
