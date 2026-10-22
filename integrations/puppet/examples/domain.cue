package puppetpilot

// Every value in this file is fictional and exists only to exercise the model.
#EnvironmentID: "dev" | "preprod" | "prod"
#DatacenterID:  "dc-a" | "dc-b"

#DomainRegistry: {
	environments: close({
		dev: {
			displayName:    "Development"
			promotionOrder: 0
		}
		preprod: {
			displayName:    "Pre-production"
			promotionOrder: 1
		}
		prod: {
			displayName:    "Production"
			promotionOrder: 2
		}
	})
	datacenters: close({
		"dc-a": {
			id:        "dc-a"
			longName:  "Fictional North Facility"
			shortName: "north"
			environments: close({
				dev:     true
				preprod: true
			})
		}
		"dc-b": {
			id:        "dc-b"
			longName:  "Fictional South Facility"
			shortName: "south"
			environments: close({
				preprod: true
				prod:    true
			})
		}
	})
}

domainRegistry: #DomainRegistry

#ModeByEnvironment: {
	dev:     "development"
	preprod: "preproduction"
	prod:    "production"
}

#NodeIdentity: {
	hostname:    string & !=""
	datacenter:  #DatacenterID
	environment: #EnvironmentID
	let dc = datacenter
	let env = environment

	// Supplied by the canonical hostname parser. The equalities below check
	// consistency without duplicating a production hostname grammar in CUE.
	hostnameIdentity: {
		parser:      string & !=""
		datacenter:  dc
		environment: env
	}
	web?: _

	_datacenterRecord:   domainRegistry.datacenters[datacenter]
	_environmentAllowed: _datacenterRecord.environments[environment] & true
}

// A single Hiera source can legitimately omit values supplied elsewhere.
#SourceHieraFragment: {
	hostname?:    string
	datacenter?:  #DatacenterID
	environment?: #EnvironmentID
	web?: {
		enabled?: bool
		settings?: {
			mode?:     string
			logLevel?: string
		}
	}
}

// This illustrative map is closed: an unexpected setting is an error.
#WebSettings: close({
	mode:     "development" | "preproduction" | "production"
	logLevel: "debug" | "info" | "warn"
})

#EffectiveNodeData: #NodeIdentity & {
	environment: #EnvironmentID
	let env = environment
	web: {
		enabled: bool
		settings: #WebSettings & {
			mode: #ModeByEnvironment[env]
		}
	}
}
