module github.com/mbauer83/effect-golang-schema

go 1.27.0

require (
	// A protobuf compiler and the reference implementation, used only by the
	// tests: the proto3 projection is compiled by a real compiler and the wire
	// codec is checked against the canonical encoder in both directions.
	// Nothing in the module itself needs either, which is what keeps this
	// layer free of a dependency every user would acquire.
	github.com/bufbuild/protocompile v0.14.1
	github.com/mbauer83/effect-golang v0.0.0
	// A JSON Schema validator, used only by the tests: the projection is
	// checked against a parser that has never seen this module.
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3
	google.golang.org/protobuf v1.36.12
)

require (
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

// effect-golang is not published yet, so it is resolved from a sibling working
// copy. A replace is ignored by anything that depends on *this* module, so it
// is a development arrangement and not a distribution one: replace both with a
// version requirement once the runtime is tagged.
replace github.com/mbauer83/effect-golang => ../effect-golang
