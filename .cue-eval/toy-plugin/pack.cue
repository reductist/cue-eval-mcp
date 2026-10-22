package toyplugin

#PackFormatVersion: "cue-eval.dev/pack/v1alpha1"

#ContractPackManifest: {
	apiVersion: #PackFormatVersion
	id:         string & =~"^[a-z0-9][a-z0-9._-]{0,63}$"
	review: {
		state:           "candidate" | "reviewed"
		reviewedDigest?: string & =~"^sha256:[0-9a-f]{64}$"
	}
	targets: [string]: {
		definition: string & =~"^#[A-Za-z_][A-Za-z0-9_]*$"
		artifacts: [...string]
		formats: [...("json" | "yaml" | "cue")]
	}
	mappings: [...{
		artifactKind: string & !=""
		target:       string & =~"^#[A-Za-z_][A-Za-z0-9_]*$"
	}]
}

pack: #ContractPackManifest & {
	apiVersion: #PackFormatVersion
	id:         "toy-plugin"

	// This draft is intentionally not asserted to be reviewed. A future digest
	// will identify reviewed bytes but will not, by itself, prove approval.
	review: state: "candidate"

	targets: {
		manifest: {
			definition: "#PluginManifest"
			artifacts: ["plugin-manifest"]
			formats: ["json", "yaml", "cue"]
		}
		output: {
			definition: "#PluginOutput"
			artifacts: ["plugin-output"]
			formats: ["json", "yaml", "cue"]
		}
	}

	mappings: [
		{artifactKind: "plugin-manifest", target: "#PluginManifest"},
		{artifactKind: "plugin-output", target: "#PluginOutput"},
	]
}
