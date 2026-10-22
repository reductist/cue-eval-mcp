package api

#Digest: string & =~"^sha256:[0-9a-f]{64}$"

#Diagnostic: {
	code: string & !=""
	severity: "error" | "warning" | "info"
	phase: "request" | "load" | "resolve" | "decode" | "unify" | "validate" | "limit" | "internal"
	message: string & !=""
	path?: string
	source?: {
		name: string
		line?: int & >=1
		column?: int & >=1
	}
}

#Provenance: {
	workspaceRoot: string
	contractFiles: [...string] | null
	subjectSource: string
	declaredReviewState: "" | "candidate" | "reviewed"
	reviewVerified: false
	reviewLimitation?: string
}

#ResolvedProvenance: #Provenance & {
	workspaceRoot: "."
	contractFiles: [...string]
	declaredReviewState: "candidate" | "reviewed"
}

#ValueOrigins: {
	suppliedPaths?:        [...string]
	missingEvidencePaths?: [...string]
}

#ItemResultBase: {
	id: string
	mode: string
	pack: string
	target: string
	outcome: "valid-concrete" | "invalid-concrete" | "compatible-so-far" | "conflict" | "error"
	gateEligible: false
	completeCheck: bool
	contractDigest?: #Digest
	inputDigest?: #Digest
	scope?: string
	provenance: #Provenance
	valueOrigins?: #ValueOrigins
	diagnostics: [...#Diagnostic]
	diagnosticsTruncated: bool

}

#AuthorResult: #ItemResultBase & {
	id: string & !=""
	mode: "author"
	pack: string & !=""
	target: string & !=""
	outcome: "compatible-so-far" | "conflict"
	completeCheck: false
	contractDigest: #Digest
	inputDigest: #Digest
	scope: string & !=""
	provenance: #ResolvedProvenance
}

#ImplementationResult: #ItemResultBase & {
	id: string & !=""
	mode: "implementation"
	pack: string & !=""
	target: string & !=""
	outcome: "valid-concrete" | "invalid-concrete"
	completeCheck: true
	contractDigest: #Digest
	inputDigest: #Digest
	scope: string & !=""
	provenance: #ResolvedProvenance
}

// Request, resolution, load, decode, limit, and internal failures can occur
// before a pack or subject has been resolved, so their provenance may be empty.
#ErrorItemResult: #ItemResultBase & {
	outcome: "error"
	completeCheck: false
}

#ItemResult: #AuthorResult | #ImplementationResult | #ErrorItemResult

#EvaluateResult: {
	apiVersion: #APIVersion
	engine: {
		kitVersion: string & !=""
		cueVersion: string & !=""
	}
	limits: {
		maxBatchItems: int & >0
		maxInputBytes: int & >0
		maxContractBytes: int & >0
		maxContractFiles: int & >0
		maxDiagnostics: int & >0
		maxDiagnosticBytes: int & >0
	}
	diagnostics?: [...#Diagnostic]
	items: [...#ItemResult]
}
