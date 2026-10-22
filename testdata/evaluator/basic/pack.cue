package basic

pack: {
	apiVersion: "cue-eval.dev/pack/v1alpha1"
	id:         "evaluator-basic"
	review: state: "candidate"
	targets: record: {
		definition: "#Record"
		artifacts: ["test-record"]
		formats: ["json", "yaml", "cue"]
	}
	mappings: [{artifactKind: "test-record", target: "#Record"}]
}
