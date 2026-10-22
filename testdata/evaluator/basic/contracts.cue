package basic

// #Record exercises recursive completeness and finalization of defaults.
#Record: close({
	id: string & =~"^[a-z][a-z0-9-]{1,15}$"
	nested: {
		count: int & >=1 & <=10
	}
	policy: *"standard" | "restricted"
})

// #StrictEvidence exercises evidence-presence checks independently of value
// validation. Every materialized value must be present in implementation input.
#StrictEvidence: close({
	id:         string
	apiVersion: "example.test/v1"
	derivedID:  id
	policy:     *"standard" | "restricted"
	optional?:  string
	nullable:   string | null
	nested: close({
		mode: "safe"
		items: [...close({
			name: string
			port: *80 | int
		})]
	})
	large:   9007199254740993
	precise: 0.1000000000000000001
	"a.b":   "literal"
	a: close({
		b: "nested"
	})
})
