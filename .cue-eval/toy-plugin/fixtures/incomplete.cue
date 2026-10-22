package toyplugin

// This candidate deliberately omits capabilities and a lower timeout bound.
// Its constraints still have a non-empty intersection with #PluginManifest.
#CandidateManifest: {
	apiVersion: "toy.dev/v1"
	name:       string
	timeoutMs:  int & <=3000
}
