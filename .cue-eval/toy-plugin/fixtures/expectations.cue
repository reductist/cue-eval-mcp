package toyplugin

// Machine-readable expected semantics for a future golden-fixture runner.
expectations: {
	"positive.json": {
		evaluation:    "concrete-data"
		outcome:       "valid-concrete"
		completeCheck: true
		// Conditional: only after this pack crosses a trusted review boundary.
		gateEligibleWithReviewedBaseline: true
	}
	"negative.json": {
		evaluation:                       "concrete-data"
		outcome:                          "invalid-concrete"
		completeCheck:                    true
		gateEligibleWithReviewedBaseline: false
	}
	"incomplete.cue": {
		evaluation:                       "partial-schema"
		outcome:                          "compatible-so-far"
		completeCheck:                    false
		gateEligibleWithReviewedBaseline: false
	}
	"conflicting.cue": {
		evaluation:                       "partial-schema"
		outcome:                          "conflict"
		completeCheck:                    false
		gateEligibleWithReviewedBaseline: false
	}
}
