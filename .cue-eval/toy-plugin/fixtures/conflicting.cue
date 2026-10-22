package toyplugin

// Both fields conflict with the selected baseline shape in #PluginManifest.
#CandidateManifest: {
	apiVersion: "toy.dev/v2"
	timeoutMs:  int & >=6000
}
