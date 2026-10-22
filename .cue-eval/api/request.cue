package api

#APIVersion: "cue-eval.dev/v1alpha1"
#Identifier: string & =~"^[a-z0-9][a-z0-9._-]{0,63}$"
#Target: string & =~"^(#[A-Za-z_][A-Za-z0-9_]*|[a-z0-9][a-z0-9._-]{0,63})$"

#InlineSource: {
	type: "inline"
	content: string
	path?: _|_
}

#FileSource: {
	type: "file"
	path: string & !=""
	content?: _|_
}

#Source: #InlineSource | #FileSource

#EvaluateItemBase: {
	id: string & !=""
	mode: "author" | "implementation"
	pack: #Identifier
	target: #Target
	format: "json" | "yaml" | "cue"
	expression?: string
	source: #Source
}

#AuthorItem: #EvaluateItemBase & {
	mode: "author"
	format: "cue"
	expression: string & =~"^#[A-Za-z_][A-Za-z0-9_]*$"
}

#ImplementationItem: #EvaluateItemBase & {
	mode: "implementation"
	format: "json" | "yaml"
	expression?: _|_
}

#EvaluateItem: #AuthorItem | #ImplementationItem

// Trust state is intentionally absent. The workspace loader derives declared
// review metadata, while an independent verifier is required for gate trust.
#EvaluateRequest: {
	apiVersion?: #APIVersion
	// The runtime also enforces its configured maximum batch size (32 by
	// default); the schema guarantees at least one item.
	items: [#EvaluateItem, ...#EvaluateItem]
}
