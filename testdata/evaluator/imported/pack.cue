package imported

pack: {
	apiVersion: "cue-eval.dev/pack/v1alpha1"
	id:         "evaluator-imported"
	review: state: "candidate"
	targets: record: {
		definition: "#Record"
		artifacts: ["test-record"]
		formats: ["json"]
	}
	mappings: [{artifactKind: "test-record", target: "#Record"}]
}
