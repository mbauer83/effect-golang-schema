package schema

// A schema whose names are spelled one way whatever format carries it.

import (
	"github.com/mbauer83/effect-golang-schema/schema/dynamic"
	"github.com/mbauer83/effect-golang-schema/schema/naming"
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// WithNaming is this schema with every member name spelled by strategy, in every
// format: its description is respelled, and whatever sink or source it meets
// is told to spell names that way, so nested objects and tagged unions follow.
//
// It is how a mapping fixes the names of what it stores -- a table's columns
// are snake_case whoever reads them -- where a format's own strategy, such as
// MemberNaming, is chosen by the reader. A name given exactly with Rename stays
// as it was given.
//
// Projections are made before spelling: the spelled schema is the end of a
// chain, and Omit, Rename or Represent on it are refused.
func (schema Schema[A]) WithNaming(strategy naming.Strategy) Schema[A] {
	if strategy.IsLiteral() {
		return schema
	}
	if fault := Validate(schema); fault != nil {
		return faultySchema[A](structure.Spell(schema.node, strategy), fault)
	}
	inner := schema
	return of(
		structure.Spell(schema.node, strategy),
		func(value A, into Sink) error {
			return Encode(inner, value, &namingSink{Sink: into, strategy: strategy})
		},
		func(from Source) (A, error) {
			return Decode(inner, &namingSource{Source: from, strategy: strategy})
		},
	)
}

// namingSink is a sink that spells names by strategy, whatever the sink
// beneath it would have done.
type namingSink struct {
	Sink
	strategy naming.Strategy
}

func (sink *namingSink) NamingStrategy() naming.Strategy { return sink.strategy }

// namingSource is a source whose names are spelled by strategy. It buffers a
// value when the source beneath it can, so a tagged union can still be read
// through it.
type namingSource struct {
	Source
	strategy naming.Strategy
}

func (source *namingSource) NamingStrategy() naming.Strategy { return source.strategy }

func (source *namingSource) Buffer() (dynamic.Value, error) {
	bufferer, can := source.Source.(Bufferer)
	if !can {
		return nil, fail(
			"this format cannot read a union told apart by a field, because the "+
				"name may arrive after the fields it settles", nil)
	}
	return bufferer.Buffer()
}
