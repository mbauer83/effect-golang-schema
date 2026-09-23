package schema

// Applying the constraints a description recorded, using the combinators they
// came from.
//
// There is one function per kind rather than one generic function, for the
// reason the constraints themselves are per kind: measuring a value is
// type-dependent, and Go cannot dispatch a generic call on what A happens to
// be. Each of these is called from a place that knows the type, which is what
// makes reusing the real constraint possible -- and reusing it is what keeps
// one opinion about what a shape admits.

import (
	"math"

	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

func constrainText(inner Schema[string], constraints []structure.Constraint) Schema[string] {
	for _, constraint := range constraints {
		switch narrowed := constraint.(type) {
		case structure.MinLength:
			inner = inner.Check(MinLength(narrowed.Value))
		case structure.MaxLength:
			inner = inner.Check(MaxLength(narrowed.Value))
		case structure.Pattern:
			inner = inner.Check(Pattern(narrowed.Expression))
		default:
			return faultySchema[string](inner.node, constraintMismatch("text"))
		}
	}
	return inner
}

func constrainInteger(inner Schema[int64], constraints []structure.Constraint) Schema[int64] {
	for _, constraint := range constraints {
		switch narrowed := constraint.(type) {
		case structure.AtLeast:
			inner = inner.Check(AtLeast(wireBound(narrowed.Value)))
		case structure.AtMost:
			inner = inner.Check(AtMost(wireBound(narrowed.Value)))
		case structure.Above:
			inner = inner.Check(Above(wireBound(narrowed.Value)))
		case structure.Below:
			inner = inner.Check(Below(wireBound(narrowed.Value)))
		default:
			return faultySchema[int64](inner.node, constraintMismatch("a whole number"))
		}
	}
	return inner
}

func constrainNumber(inner Schema[float64], constraints []structure.Constraint) Schema[float64] {
	for _, constraint := range constraints {
		switch narrowed := constraint.(type) {
		case structure.AtLeast:
			inner = inner.Check(AtLeast(narrowed.Value))
		case structure.AtMost:
			inner = inner.Check(AtMost(narrowed.Value))
		case structure.Above:
			inner = inner.Check(Above(narrowed.Value))
		case structure.Below:
			inner = inner.Check(Below(narrowed.Value))
		default:
			return faultySchema[float64](inner.node, constraintMismatch("a number"))
		}
	}
	return inner
}

func constrainList[A any](inner Schema[[]A], constraints []structure.Constraint) Schema[[]A] {
	for _, constraint := range constraints {
		switch narrowed := constraint.(type) {
		case structure.MinItems:
			inner = inner.Check(MinItems[A](narrowed.Value))
		case structure.MaxItems:
			inner = inner.Check(MaxItems[A](narrowed.Value))
		default:
			return faultySchema[[]A](inner.node, constraintMismatch("a list"))
		}
	}
	return inner
}

// wireBound converts a recorded bound back to the width the wire carries,
// saturating rather than wrapping.
//
// A bound is recorded as a float64, which cannot hold the largest int64
// exactly: the nearest float to MaxInt64 is one above it. Saturating keeps the
// meaning -- a bound at the edge of the range means "no further" -- where
// converting would overflow into a negative number.
func wireBound(bound float64) int64 {
	switch {
	case bound >= math.MaxInt64:
		return math.MaxInt64
	case bound <= math.MinInt64:
		return math.MinInt64
	default:
		return int64(bound)
	}
}

func constraintMismatch(kind string) error {
	return fail("carries a constraint that does not narrow "+kind, nil)
}
