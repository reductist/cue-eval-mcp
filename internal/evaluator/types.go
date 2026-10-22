package evaluator

const (
	APIVersion     = "cue-eval.dev/v1alpha1"
	ProjectVersion = "cue-eval.dev/project/v1alpha1"
	KitVersion     = "0.1.0-dev"
)

type Limits struct {
	MaxBatchItems      int   `json:"maxBatchItems"`
	MaxInputBytes      int64 `json:"maxInputBytes"`
	MaxContractBytes   int64 `json:"maxContractBytes"`
	MaxContractFiles   int   `json:"maxContractFiles"`
	MaxDiagnostics     int   `json:"maxDiagnostics"`
	MaxDiagnosticBytes int   `json:"maxDiagnosticBytes"`
}

func DefaultLimits() Limits { return Limits{32, 1 << 20, 4 << 20, 256, 50, 1024} }

type Review struct {
	State string `json:"state"`
}
type Target struct {
	Name          string   `json:"name"`
	Definition    string   `json:"definition"`
	ArtifactKinds []string `json:"artifactKinds,omitempty"`
	Formats       []string `json:"formats"`
}
type Pack struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	Root       string   `json:"root"`
	Review     Review   `json:"review"`
	Sources    []string `json:"sources"`
	InputRoots []string `json:"inputRoots,omitempty"`
	Targets    []Target `json:"targets"`
}
type Project struct {
	APIVersion string `json:"apiVersion"`
	Packs      []Pack `json:"packs"`
}
type Source struct {
	Type    string `json:"type"`
	Path    string `json:"path,omitempty"`
	Content string `json:"content,omitempty"`
}
type DiscoverRequest struct {
	Query string `json:"query,omitempty"`
	Limit int    `json:"limit,omitempty"`
}
type PackSummary struct {
	ID                  string   `json:"id"`
	Title               string   `json:"title"`
	Summary             string   `json:"summary"`
	DeclaredReviewState string   `json:"declaredReviewState"`
	ReviewVerified      bool     `json:"reviewVerified"`
	ContractDigest      string   `json:"contractDigest"`
	Targets             []Target `json:"targets"`
}
type DiscoverResult struct {
	APIVersion string        `json:"apiVersion"`
	Packs      []PackSummary `json:"packs"`
	Truncated  bool          `json:"truncated"`
}
type InspectRequest struct {
	Pack          string `json:"pack"`
	Target        string `json:"target,omitempty"`
	IncludeSource bool   `json:"includeSource,omitempty"`
}
type SourceFile struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
	Bytes  int    `json:"bytes"`
	Source string `json:"source,omitempty"`
}
type InspectResult struct {
	APIVersion     string       `json:"apiVersion"`
	Pack           PackSummary  `json:"pack"`
	SelectedTarget *Target      `json:"selectedTarget,omitempty"`
	Sources        []SourceFile `json:"sources"`
}
type EvaluateRequest struct {
	APIVersion string         `json:"apiVersion,omitempty"`
	Items      []EvaluateItem `json:"items"`
}
type EvaluateItem struct {
	ID         string `json:"id"`
	Mode       string `json:"mode"`
	Pack       string `json:"pack"`
	Target     string `json:"target"`
	Format     string `json:"format"`
	Expression string `json:"expression,omitempty"`
	Source     Source `json:"source"`
}
type Diagnostic struct {
	Code     string          `json:"code"`
	Severity string          `json:"severity"`
	Phase    string          `json:"phase"`
	Message  string          `json:"message"`
	Path     string          `json:"path,omitempty"`
	Source   *DiagnosticSite `json:"source,omitempty"`
}
type DiagnosticSite struct {
	Name   string `json:"name"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
}
type Provenance struct {
	WorkspaceRoot       string   `json:"workspaceRoot"`
	ContractFiles       []string `json:"contractFiles"`
	SubjectSource       string   `json:"subjectSource"`
	DeclaredReviewState string   `json:"declaredReviewState"`
	ReviewVerified      bool     `json:"reviewVerified"`
	ReviewLimitation    string   `json:"reviewLimitation,omitempty"`
}
type ValueOrigins struct {
	SuppliedPaths        []string `json:"suppliedPaths,omitempty"`
	MissingEvidencePaths []string `json:"missingEvidencePaths,omitempty"`
}
type ItemResult struct {
	ID                   string       `json:"id"`
	Mode                 string       `json:"mode"`
	Pack                 string       `json:"pack"`
	Target               string       `json:"target"`
	Outcome              string       `json:"outcome"`
	GateEligible         bool         `json:"gateEligible"`
	CompleteCheck        bool         `json:"completeCheck"`
	ContractDigest       string       `json:"contractDigest,omitempty"`
	InputDigest          string       `json:"inputDigest,omitempty"`
	Scope                string       `json:"scope,omitempty"`
	Provenance           Provenance   `json:"provenance"`
	ValueOrigins         ValueOrigins `json:"valueOrigins,omitempty"`
	Diagnostics          []Diagnostic `json:"diagnostics"`
	DiagnosticsTruncated bool         `json:"diagnosticsTruncated"`
}
type EvaluateResult struct {
	APIVersion  string         `json:"apiVersion"`
	Engine      EngineIdentity `json:"engine"`
	Limits      Limits         `json:"limits"`
	Diagnostics []Diagnostic   `json:"diagnostics,omitempty"`
	Items       []ItemResult   `json:"items"`
}
type EngineIdentity struct {
	KitVersion string `json:"kitVersion"`
	CUEVersion string `json:"cueVersion"`
}
