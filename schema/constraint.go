package schema

// A constraint narrows what a shape admits and says so in the description, so
// one declaration both refuses a value and appears in the published contract.
// Enforcement happens on encode as well as decode: a value this program built
// that breaks its own constraint is a mistake here, and finding it at the
// boundary is better than sending it.
//
// A constraint is a value and is applied by a method, which is the same rule
// Named and Documented follow -- a modifier changes a schema rather than
// building one, and a reader should not have to remember which of them wrap
// and which are called on what they change. Written as wrappers, two bounds on
// one string read inside out:
//
//	MaxLength(MinLength(Text(), 1), 32)     // the subject is in the middle
//	Text().Constrained(MinLength(1), MaxLength(32))
//
// Each constraint is its own constructor rather than one generic Constrained
// taking a measuring function, because measuring a value is type-dependent --
// a number is compared, a string is counted, a list is counted differently --
// and a generic one would have to take a function nobody wants to write.
//
// The type parameter is what keeps a misapplication a compile error: MinLength
// is a Constraint[string] and Text() is a Schema[string], so a length on a
// number does not build. It cannot be done with a method per constraint
// instead: Go has no way to declare a method for particular instantiations of
// a generic type -- the receiver's type parameters are declarations, so
// `func (Schema[string]) MinLength(int)` silently declares a parameter *named*
// string and lands the method on every schema -- and a method cannot narrow
// the receiver's constraint either.

import (
	"strconv"

	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// Constraint is a narrowing of what a shape admits, of the type it admits.
//
// The same word as structure.Constraint and deliberately: that is the
// narrowing as a description carries it, and this is the narrowing as this
// package applies it -- the description, plus the check that enforces it, plus
// the type it is about. One concept, two levels, the way Schema is to
// structure.Node.
type Constraint[A any] struct {
	described structure.Constraint
	check     func(A) error
	fault     error
}

// Constrained is this schema, admitting only what all of those admit.
//
// In the order given, so a value that breaks two of them is reported against
// the first -- which is the one a reader of the declaration meets first.
func (schema Schema[A]) Constrained(constraints ...Constraint[A]) Schema[A] {
	narrowed := schema
	for _, constraint := range constraints {
		if constraint.fault != nil {
			return faultedSchema[A](schema.node, constraint.fault)
		}
		narrowed = applyConstraint(narrowed, constraint.described, constraint.check)
	}
	return narrowed
}

func newConstraint[A any](described structure.Constraint, check func(A) error) Constraint[A] {
	return Constraint[A]{described: described, check: check}
}

// applyConstraint records the constraint in the description and checks it in both
// directions.
func applyConstraint[A any](
	inner Schema[A],
	constraint structure.Constraint,
	check func(A) error,
) Schema[A] {
	if fault := Validate(inner); fault != nil {
		return faultedSchema[A](inner.node, fault)
	}
	node, applies := withConstraint(inner.node, constraint)
	if !applies {
		return faultedSchema[A](inner.node,
			fail("a constraint applies to a scalar or a list, and this is neither", nil))
	}
	return of(
		node,
		func(value A, into Sink) error {
			if err := check(value); err != nil {
				return err
			}
			return Encode(inner, value, into)
		},
		func(from Source) (A, error) {
			decoded, err := Decode(inner, from)
			if err != nil {
				return decoded, err
			}
			if err := check(decoded); err != nil {
				var missing A
				return missing, err
			}
			return decoded, nil
		},
	)
}

// withConstraint attaches a constraint to the shape it belongs to.
//
// A constraint reaches through a refinement, because a refinement keeps the
// shape it was derived from and a bound on that shape is still a bound. It does
// not reach through an object or a union: a constraint on one of those would
// have to say which member it meant.
func withConstraint(node structure.Node, constraint structure.Constraint) (structure.Node, bool) {
	switch shape := node.(type) {
	case structure.Scalar:
		shape.Constraints = append(append([]structure.Constraint{}, shape.Constraints...), constraint)
		return shape, true
	case structure.Sequence:
		shape.Constraints = append(append([]structure.Constraint{}, shape.Constraints...), constraint)
		return shape, true
	default:
		return node, false
	}
}

// number renders a bound the way a message should read: 1 rather than 1e+00.
func number[A numeric](value A) string {
	return strconv.FormatFloat(float64(value), 'g', -1, 64)
}

// length counts characters rather than bytes, so a constraint on a string means
// what a reader of the contract would expect it to mean.
func length(value string) int {
	return len([]rune(value))
}
