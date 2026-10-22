package evaluator

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/parser"
	"cuelang.org/go/encoding/yaml"
)

var (
	idPattern         = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
	definitionPattern = regexp.MustCompile(`^#[A-Za-z_][A-Za-z0-9_]*$`)
)

type Engine struct{ workspace *Workspace }

func New(workspace *Workspace) *Engine { return &Engine{workspace: workspace} }

func (e *Engine) loadProject() (Project, error) {
	b, err := e.workspace.readFile(e.workspace.ManifestPath, e.workspace.Limits.MaxContractBytes)
	if err != nil {
		return Project{}, err
	}
	var p Project
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		return Project{}, fmt.Errorf("decode project manifest: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Project{}, errors.New("decode project manifest: trailing JSON value")
		}
		return Project{}, fmt.Errorf("decode project manifest trailing data: %w", err)
	}
	if p.APIVersion != ProjectVersion {
		return Project{}, fmt.Errorf("unsupported project apiVersion %q", p.APIVersion)
	}
	seen := map[string]bool{}
	for i := range p.Packs {
		pack := &p.Packs[i]
		if len(pack.ID) > 64 || len(pack.Title) > 256 || len(pack.Summary) > 1024 || len(pack.Root) > 4096 {
			return Project{}, fmt.Errorf("pack metadata exceeds limits")
		}
		if !idPattern.MatchString(pack.ID) || seen[pack.ID] {
			return Project{}, fmt.Errorf("invalid or duplicate pack id %q", pack.ID)
		}
		seen[pack.ID] = true
		if pack.Review.State != "candidate" && pack.Review.State != "reviewed" {
			return Project{}, fmt.Errorf("invalid declared review state for %q", pack.ID)
		}
		if len(pack.Sources) == 0 || len(pack.Sources) > e.workspace.Limits.MaxContractFiles {
			return Project{}, fmt.Errorf("invalid source count for %q", pack.ID)
		}
		if pack.Title == "" || pack.Summary == "" || pack.Root == "" || len(pack.Targets) == 0 {
			return Project{}, fmt.Errorf("pack %q is missing required metadata", pack.ID)
		}
		for _, source := range pack.Sources {
			if filepath.Ext(source) != ".cue" {
				return Project{}, fmt.Errorf("pack %q source is not CUE", pack.ID)
			}
			if _, err := e.workspace.ResolveFile(filepath.Join(pack.Root, source)); err != nil {
				return Project{}, fmt.Errorf("pack %q: %w", pack.ID, err)
			}
		}
		for _, root := range pack.InputRoots {
			clean := filepath.Clean(root)
			if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
				return Project{}, fmt.Errorf("pack %q has invalid input root", pack.ID)
			}
		}
		targets := map[string]bool{}
		for _, t := range pack.Targets {
			if !idPattern.MatchString(t.Name) || !definitionPattern.MatchString(t.Definition) || targets[t.Name] || len(t.Formats) == 0 {
				return Project{}, fmt.Errorf("invalid target in pack %q", pack.ID)
			}
			targets[t.Name] = true
			for _, f := range t.Formats {
				if f != "json" && f != "yaml" && f != "cue" {
					return Project{}, fmt.Errorf("invalid target format in pack %q", pack.ID)
				}
			}
		}
	}
	sort.Slice(p.Packs, func(i, j int) bool { return p.Packs[i].ID < p.Packs[j].ID })
	return p, nil
}

func (e *Engine) Discover(ctx context.Context, req DiscoverRequest) (DiscoverResult, error) {
	if err := ctx.Err(); err != nil {
		return DiscoverResult{}, err
	}
	p, err := e.loadProject()
	if err != nil {
		return DiscoverResult{}, err
	}
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	query := strings.ToLower(strings.TrimSpace(req.Query))
	result := DiscoverResult{APIVersion: APIVersion, Packs: []PackSummary{}}
	for _, pack := range p.Packs {
		if query != "" && !strings.Contains(strings.ToLower(pack.ID+" "+pack.Title+" "+pack.Summary), query) {
			continue
		}
		if len(result.Packs) == limit {
			result.Truncated = true
			break
		}
		summary, _, err := e.packSummary(pack, false)
		if err != nil {
			return DiscoverResult{}, err
		}
		result.Packs = append(result.Packs, summary)
	}
	return result, nil
}

func (e *Engine) Inspect(ctx context.Context, req InspectRequest) (InspectResult, error) {
	if err := ctx.Err(); err != nil {
		return InspectResult{}, err
	}
	pack, err := e.findPack(req.Pack)
	if err != nil {
		return InspectResult{}, err
	}
	summary, files, err := e.packSummary(pack, req.IncludeSource)
	if err != nil {
		return InspectResult{}, err
	}
	result := InspectResult{APIVersion: APIVersion, Pack: summary, Sources: files}
	if req.Target != "" {
		t, err := findTarget(pack, req.Target)
		if err != nil {
			return InspectResult{}, err
		}
		result.SelectedTarget = &t
	}
	return result, nil
}

func (e *Engine) Evaluate(ctx context.Context, req EvaluateRequest) EvaluateResult {
	result := EvaluateResult{APIVersion: APIVersion, Engine: EngineIdentity{KitVersion: KitVersion, CUEVersion: cue.LanguageVersion()}, Limits: e.workspace.Limits, Items: []ItemResult{}}
	if req.APIVersion != "" && req.APIVersion != APIVersion {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{Code: "unsupported-api-version", Severity: "error", Phase: "request", Message: "request apiVersion is not supported"})
		return result
	}
	if len(req.Items) == 0 || len(req.Items) > e.workspace.Limits.MaxBatchItems {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{Code: "batch-limit", Severity: "error", Phase: "limit", Message: "batch item count is outside the configured limit"})
		return result
	}
	for _, item := range req.Items {
		if err := ctx.Err(); err != nil {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{Code: "cancelled", Severity: "error", Phase: "internal", Message: "evaluation was cancelled before all items completed"})
			break
		}
		result.Items = append(result.Items, e.evaluateItem(item))
	}
	return result
}

func (e *Engine) evaluateItem(item EvaluateItem) ItemResult {
	r := ItemResult{ID: item.ID, Mode: item.Mode, Pack: item.Pack, Target: item.Target, Outcome: "error", GateEligible: false, Diagnostics: []Diagnostic{}}
	if len(item.ID) > 128 || len(item.Pack) > 64 || len(item.Target) > 128 || len(item.Format) > 16 || len(item.Expression) > 128 || len(item.Source.Path) > 4096 {
		return withDiagnostic(r, "metadata-limit", "limit", "request metadata exceeds configured limits")
	}
	if item.ID == "" || (item.Mode != "author" && item.Mode != "implementation") {
		return withDiagnostic(r, "invalid-request", "request", "item id or mode is invalid")
	}
	pack, err := e.findPack(item.Pack)
	if err != nil {
		return withDiagnostic(r, "pack-not-found", "resolve", "contract pack could not be resolved")
	}
	target, err := findTarget(pack, item.Target)
	if err != nil {
		return withDiagnostic(r, "target-not-found", "resolve", "contract target could not be resolved")
	}
	if !contains(target.Formats, item.Format) {
		return withDiagnostic(r, "unsupported-format", "request", "subject format is not allowed by the target")
	}
	contract, cuectx, summary, files, err := e.compilePack(pack)
	r.ContractDigest = summary.ContractDigest
	r.Scope = pack.ID + ":" + target.Definition
	r.Provenance = Provenance{WorkspaceRoot: ".", DeclaredReviewState: pack.Review.State, ReviewVerified: false, ReviewLimitation: "review metadata is declared by the project manifest; no independent verifier is configured"}
	for _, f := range files {
		r.Provenance.ContractFiles = append(r.Provenance.ContractFiles, f.Path)
	}
	if err != nil {
		return appendError(r, err, "contract-load", "load")
	}
	schema := contract.LookupPath(cue.ParsePath(target.Definition))
	if !schema.Exists() || schema.Err() != nil {
		return withDiagnostic(r, "target-missing", "resolve", "contract target definition is unavailable")
	}
	bytes, sourceName, err := e.readSubject(pack, item.Source)
	r.Provenance.SubjectSource = sourceName
	if err != nil {
		return appendError(r, err, "subject-load", "load")
	}
	r.InputDigest = digest(bytes)
	var subject cue.Value
	var unified cue.Value
	var supplied map[string]string
	if item.Mode == "author" {
		if item.Format != "cue" || item.Expression == "" {
			return withDiagnostic(r, "invalid-author-subject", "request", "author mode requires a CUE source and expression")
		}
		unified, err = e.compileAuthorValue(pack, target.Definition, bytes, sourceName, item.Expression)
	} else {
		if item.Format == "cue" {
			return withDiagnostic(r, "unsupported-implementation-format", "request", "implementation mode requires JSON or YAML evidence; use CUE sources in author mode")
		}
		subject, supplied, err = decodeSubject(cuectx, bytes, sourceName, item.Format, item.Expression)
		if err == nil {
			unified = schema.Unify(subject)
		}
	}
	if err != nil {
		return appendError(r, err, "subject-decode", "decode")
	}
	if item.Mode == "author" {
		r.CompleteCheck = false
		if err := unified.Validate(); err != nil {
			r.Outcome = "conflict"
			return e.appendCUE(r, err, "contract-conflict", "unify")
		}
		r.Outcome = "compatible-so-far"
		return r
	}
	r.CompleteCheck = true
	if err := subject.Validate(cue.Final(), cue.Concrete(true)); err != nil {
		r.Outcome = "invalid-concrete"
		return e.appendCUE(r, err, "non-concrete-evidence", "validate")
	}
	if err := unified.Validate(cue.Final(), cue.Concrete(true)); err != nil {
		r.Outcome = "invalid-concrete"
		return e.appendCUE(r, err, "invalid-concrete", "validate")
	}
	materialized := map[string]string{}
	collectValuePaths(unified, nil, materialized)
	missing := missingPaths(materialized, supplied)
	if len(missing) != 0 {
		r.Outcome = "invalid-concrete"
		r.ValueOrigins = ValueOrigins{SuppliedPaths: sortedPathDisplays(supplied), MissingEvidencePaths: missing}
		for _, path := range missing {
			if len(r.Diagnostics) >= e.workspace.Limits.MaxDiagnostics {
				r.DiagnosticsTruncated = true
				break
			}
			path, truncated := truncateString(path, e.workspace.Limits.MaxDiagnosticBytes)
			if truncated {
				r.DiagnosticsTruncated = true
			}
			r.Diagnostics = append(r.Diagnostics, Diagnostic{
				Code:     "missing-evidence",
				Severity: "error",
				Phase:    "validate",
				Message:  "the contract materialized a value that was absent from the supplied evidence",
				Path:     path,
			})
		}
		return r
	}
	r.Outcome = "valid-concrete"
	r.ValueOrigins = ValueOrigins{SuppliedPaths: sortedPathDisplays(supplied)}
	return r
}

func (e *Engine) findPack(id string) (Pack, error) {
	p, err := e.loadProject()
	if err != nil {
		return Pack{}, err
	}
	for _, pack := range p.Packs {
		if pack.ID == id {
			return pack, nil
		}
	}
	return Pack{}, errors.New("pack not found")
}
func findTarget(pack Pack, name string) (Target, error) {
	for _, t := range pack.Targets {
		if t.Name == name || t.Definition == name {
			return t, nil
		}
	}
	return Target{}, errors.New("target not found")
}

func (e *Engine) packSummary(pack Pack, source bool) (PackSummary, []SourceFile, error) {
	_, _, summary, files, err := e.compilePack(pack)
	if err != nil {
		return PackSummary{}, nil, err
	}
	if !source {
		for i := range files {
			files[i].Source = ""
		}
	}
	return summary, files, nil
}

func (e *Engine) compilePack(pack Pack) (cue.Value, *cue.Context, PackSummary, []SourceFile, error) {
	ctx := cuecontext.New()
	buildctx := build.NewContext()
	inst := buildctx.NewInstance(filepath.Join(e.workspace.Root, pack.Root), nil)
	h := sha256.New()
	canonical, _ := json.Marshal(pack)
	writeDigestPart(h, "project-pack.json", canonical)
	files := make([]SourceFile, 0, len(pack.Sources))
	total := int64(0)
	sources := append([]string(nil), pack.Sources...)
	sort.Strings(sources)
	for _, name := range sources {
		path, err := e.workspace.ResolveFile(filepath.Join(pack.Root, name))
		if err != nil {
			return cue.Value{}, nil, PackSummary{}, nil, err
		}
		b, err := e.workspace.readFile(e.workspace.relative(path), e.workspace.Limits.MaxContractBytes-total)
		if err != nil {
			return cue.Value{}, nil, PackSummary{}, nil, err
		}
		total += int64(len(b))
		file, err := parser.ParseFile(e.workspace.relative(path), b, parser.AllErrors)
		if err != nil {
			return cue.Value{}, nil, PackSummary{}, nil, err
		}
		if hasImports(file) {
			return cue.Value{}, nil, PackSummary{}, nil, errors.New("imports are disabled in the safe local evaluator")
		}
		if err := inst.AddSyntax(file); err != nil {
			return cue.Value{}, nil, PackSummary{}, nil, err
		}
		writeDigestPart(h, e.workspace.relative(path), b)
		files = append(files, SourceFile{Path: e.workspace.relative(path), Digest: digest(b), Bytes: len(b), Source: string(b)})
	}
	if err := inst.Complete(); err != nil {
		return cue.Value{}, nil, PackSummary{}, nil, err
	}
	value := ctx.BuildInstance(inst)
	if err := value.Err(); err != nil {
		return cue.Value{}, nil, PackSummary{}, nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	summary := PackSummary{ID: pack.ID, Title: pack.Title, Summary: pack.Summary, DeclaredReviewState: pack.Review.State, ReviewVerified: false, ContractDigest: "sha256:" + hex.EncodeToString(h.Sum(nil)), Targets: pack.Targets}
	return value, ctx, summary, files, nil
}

func (e *Engine) compileAuthorValue(pack Pack, target string, subject []byte, subjectName, expression string) (cue.Value, error) {
	if !definitionPattern.MatchString(expression) {
		return cue.Value{}, errors.New("author expression must name a top-level definition")
	}
	ctx := cuecontext.New()
	buildctx := build.NewContext()
	inst := buildctx.NewInstance(filepath.Join(e.workspace.Root, pack.Root), nil)
	sources := append([]string(nil), pack.Sources...)
	sort.Strings(sources)
	total := int64(0)
	for _, name := range sources {
		path, err := e.workspace.ResolveFile(filepath.Join(pack.Root, name))
		if err != nil {
			return cue.Value{}, err
		}
		b, err := e.workspace.readFile(e.workspace.relative(path), e.workspace.Limits.MaxContractBytes-total)
		if err != nil {
			return cue.Value{}, err
		}
		total += int64(len(b))
		file, err := parser.ParseFile(e.workspace.relative(path), b, parser.AllErrors)
		if err != nil {
			return cue.Value{}, err
		}
		if hasImports(file) {
			return cue.Value{}, errors.New("imports are disabled in the safe local evaluator")
		}
		if err := inst.AddSyntax(file); err != nil {
			return cue.Value{}, err
		}
	}
	file, err := parser.ParseFile(subjectName, subject, parser.AllErrors)
	if err != nil {
		return cue.Value{}, err
	}
	if hasImports(file) {
		return cue.Value{}, errors.New("imports are disabled in subject sources")
	}
	if err := inst.AddSyntax(file); err != nil {
		return cue.Value{}, err
	}
	if err := inst.Complete(); err != nil {
		return cue.Value{}, err
	}
	root := ctx.BuildInstance(inst)
	if err := root.Err(); err != nil {
		return cue.Value{}, err
	}
	expr, err := parser.ParseExpr("cue-eval-author-expression", target+" & "+expression)
	if err != nil {
		return cue.Value{}, err
	}
	result := ctx.BuildExpr(expr, cue.Scope(root), cue.InferBuiltins(true), cue.ImportPath(inst.ID()))
	return result, nil
}

func (e *Engine) readSubject(pack Pack, src Source) ([]byte, string, error) {
	switch src.Type {
	case "inline":
		if src.Path != "" {
			return nil, "", errors.New("inline source must not set path")
		}
		if int64(len(src.Content)) > e.workspace.Limits.MaxInputBytes {
			return nil, "", errors.New("input exceeds byte limit")
		}
		return []byte(src.Content), "inline", nil
	case "file":
		if src.Content != "" {
			return nil, "", errors.New("file source must not set content")
		}
		if !withinInputRoots(src.Path, pack.InputRoots) {
			return nil, "", errors.New("file source is outside the pack input roots")
		}
		path, err := e.workspace.ResolveFile(src.Path)
		if err != nil {
			return nil, "", err
		}
		b, err := e.workspace.readFile(e.workspace.relative(path), e.workspace.Limits.MaxInputBytes)
		return b, e.workspace.relative(path), err
	default:
		return nil, "", errors.New("source type must be inline or file")
	}
}

func decodeSubject(ctx *cue.Context, b []byte, name, format, expression string) (cue.Value, map[string]string, error) {
	var file *ast.File
	var err error
	switch format {
	case "json":
		if !json.Valid(b) {
			return cue.Value{}, nil, errors.New("malformed JSON")
		}
		file, err = parser.ParseFile(name, b, parser.AllErrors)
	case "yaml":
		file, err = yaml.Extract(name, b)
	case "cue":
		file, err = parser.ParseFile(name, b, parser.AllErrors)
	default:
		return cue.Value{}, nil, errors.New("unsupported format")
	}
	if err != nil {
		return cue.Value{}, nil, err
	}
	if hasImports(file) {
		return cue.Value{}, nil, errors.New("imports are disabled in subject sources")
	}
	value := ctx.BuildFile(file)
	if err := value.Err(); err != nil {
		return cue.Value{}, nil, err
	}
	if expression != "" {
		if format != "cue" {
			return cue.Value{}, nil, errors.New("expression is only valid for CUE subjects")
		}
		value = value.LookupPath(cue.ParsePath(expression))
		if !value.Exists() || value.Err() != nil {
			return cue.Value{}, nil, errors.New("subject expression was not found")
		}
	}
	supplied := map[string]string{}
	collectValuePaths(value, nil, supplied)
	return value, supplied, nil
}

type valuePathSegment struct {
	field *string
	index int
}

func collectValuePaths(v cue.Value, prefix []valuePathSegment, out map[string]string) {
	kind := v.IncompleteKind()
	if kind&cue.StructKind != 0 {
		it, err := v.Fields(cue.Definitions(false), cue.Hidden(false), cue.Optional(false))
		if err == nil {
			for it.Next() {
				label := it.Selector().Unquoted()
				segments := appendPathSegment(prefix, valuePathSegment{field: &label})
				out[pathKey(segments)] = displayPath(segments)
				collectValuePaths(it.Value(), segments, out)
			}
		}
		return
	}
	if kind&cue.ListKind != 0 {
		it, err := v.List()
		if err == nil {
			for i := 0; it.Next(); i++ {
				segments := appendPathSegment(prefix, valuePathSegment{index: i})
				out[pathKey(segments)] = displayPath(segments)
				collectValuePaths(it.Value(), segments, out)
			}
		}
	}
}

func appendPathSegment(prefix []valuePathSegment, segment valuePathSegment) []valuePathSegment {
	result := make([]valuePathSegment, len(prefix)+1)
	copy(result, prefix)
	result[len(prefix)] = segment
	return result
}

func pathKey(segments []valuePathSegment) string {
	var b strings.Builder
	for _, segment := range segments {
		if segment.field != nil {
			fmt.Fprintf(&b, "f%d:%s;", len(*segment.field), *segment.field)
		} else {
			fmt.Fprintf(&b, "i%d;", segment.index)
		}
	}
	return b.String()
}

func displayPath(segments []valuePathSegment) string {
	selectors := make([]cue.Selector, 0, len(segments))
	for _, segment := range segments {
		if segment.field != nil {
			selectors = append(selectors, cue.Str(*segment.field))
		} else {
			selectors = append(selectors, cue.Index(segment.index))
		}
	}
	return cue.MakePath(selectors...).String()
}

func missingPaths(materialized, supplied map[string]string) []string {
	missing := make([]string, 0)
	for key, display := range materialized {
		if _, ok := supplied[key]; !ok {
			missing = append(missing, display)
		}
	}
	sort.Strings(missing)
	return missing
}

func sortedPathDisplays(paths map[string]string) []string {
	out := make([]string, 0, len(paths))
	for _, display := range paths {
		out = append(out, display)
	}
	sort.Strings(out)
	return out
}

func hasImports(file *ast.File) bool {
	for _, decl := range file.Decls {
		if _, ok := decl.(*ast.ImportDecl); ok {
			return true
		}
	}
	return false
}
func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func withinInputRoots(name string, roots []string) bool {
	if name == "" || filepath.IsAbs(name) {
		return false
	}
	clean := filepath.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return false
	}
	for _, root := range roots {
		rel, err := filepath.Rel(filepath.Clean(root), clean)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func (e *Engine) appendCUE(r ItemResult, err error, code, phase string) ItemResult {
	for _, ce := range cueerrors.Errors(err) {
		if len(r.Diagnostics) >= e.workspace.Limits.MaxDiagnostics {
			r.DiagnosticsTruncated = true
			break
		}
		pos := ce.Position()
		path, truncated := truncateString(strings.Join(ce.Path(), "."), e.workspace.Limits.MaxDiagnosticBytes)
		d := Diagnostic{Code: code, Severity: "error", Phase: phase, Message: "CUE constraints were not satisfied", Path: path}
		if truncated {
			r.DiagnosticsTruncated = true
		}
		if pos.IsValid() {
			name, cut := truncateString(filepath.ToSlash(pos.Filename()), e.workspace.Limits.MaxDiagnosticBytes)
			if cut {
				r.DiagnosticsTruncated = true
			}
			d.Source = &DiagnosticSite{Name: name, Line: pos.Line(), Column: pos.Column()}
		}
		r.Diagnostics = append(r.Diagnostics, d)
	}
	if len(r.Diagnostics) == 0 {
		r = withDiagnostic(r, code, phase, "CUE constraints were not satisfied")
	}
	return r
}
func truncateString(s string, limit int) (string, bool) {
	if len(s) <= limit {
		return s, false
	}
	return s[:limit], true
}
func appendError(r ItemResult, _ error, code, phase string) ItemResult {
	return withDiagnostic(r, code, phase, "evaluation could not complete during "+phase)
}
func withDiagnostic(r ItemResult, code, phase, message string) ItemResult {
	r.Diagnostics = append(r.Diagnostics, Diagnostic{Code: code, Severity: "error", Phase: phase, Message: message})
	return r
}
func digest(b []byte) string { sum := sha256.Sum256(b); return "sha256:" + hex.EncodeToString(sum[:]) }
func writeDigestPart(w io.Writer, name string, b []byte) {
	var n [8]byte
	binary.BigEndian.PutUint64(n[:], uint64(len(name)))
	_, _ = w.Write(n[:])
	_, _ = io.WriteString(w, name)
	binary.BigEndian.PutUint64(n[:], uint64(len(b)))
	_, _ = w.Write(n[:])
	_, _ = w.Write(b)
}
