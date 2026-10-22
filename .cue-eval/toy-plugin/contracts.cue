package toyplugin

#Capability: "read" | "transform"

#PluginManifest: {
	apiVersion: "toy.dev/v1"
	name:       string & =~"^[a-z][a-z0-9-]{2,31}$"
	timeoutMs:  int & >=100 & <=5000
	capabilities: [...#Capability]
}

#PluginOutput: {
	plugin: string & !=""
	status: "ok" | "rejected"
	items: [...{
		id:    string & !=""
		score: number & >=0 & <=1
	}]
}
