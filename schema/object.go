package schema

// An object as a whole: the fixed set of named fields a struct is, and how one
// is written and read.
//
// The fields themselves are in object_fields.go; this is the shape they add up
// to.

import (
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// Struct describes A as a fixed set of named fields.
//
// A named struct becomes a reusable component in projections that have them, so
// a type used by ten endpoints is described once. Pass an empty name to inline
// it instead.
func Struct[A any](name string, fields ...Field[A]) Schema[A] {
	node := structure.Object{Name: name, Fields: describeFields(fields)}
	if fault := firstFieldFault(fields); fault != nil {
		return faulted[A](node, fault)
	}

	required, byName := indexFields(fields)
	return of[A](
		node,
		func(value A, into Sink) error {
			return encodeFields(value, fields, into)
		},
		func(from Source) (A, error) {
			return decodeFields[A](from, byName, required)
		},
	)
}
