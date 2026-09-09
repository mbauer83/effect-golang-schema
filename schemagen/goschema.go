package schemagen

// The schema expression a described shape becomes.
//
// It is the inverse of reading a struct: the same combinator calls, derived
// from the description rather than from a Go type. A generated schema and a
// written one stay interchangeable in both directions, which is what keeps
// there being one vocabulary to learn.

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mbauer83/effect-golang-schema/schema"
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

func schemaExpression(node structure.Node) (string, error) {
	switch shape := node.(type) {
	case structure.Scalar:
		return scalarExpression(shape)
	case structure.Sequence:
		return sequenceExpression(shape)
	case structure.Mapping:
		return wrappedExpression("schema.Map", shape.Value)
	case structure.Nullable:
		return wrappedExpression("schema.Nullable", shape.Inner)
	case structure.Object:
		return namedExpression(shape.Name, "an object")
	case structure.Union:
		return namedExpression(shape.Name, "a union")
	case structure.Reference:
		return namedExpression(shape.Name, "a reference")
	default:
		return "", errors.New("this shape has no schema expression")
	}
}

// scalarExpression names the base shape and then the constraints that narrow
// it.
//
// A known format is emitted as its own constructor, and the constraints that
// constructor already carries are not emitted again -- which is worked out by
// asking the constructor what it records, rather than by a second table that
// could fall out of step with it.
func scalarExpression(shape structure.Scalar) (string, error) {
	base, carried, holds := baseExpression(shape)
	return constrainExpression(base, holds, shape.Constraints[carried:])
}

// baseExpression is the constructor for a scalar, how many of its constraints
// that constructor already records, and the Go type it describes.
func baseExpression(shape structure.Scalar) (string, int, string) {
	if known, checked := schema.FormatConstructors[shape.Format]; checked {
		return known.Call, known.Carries, known.Holds
	}
	if shape.Format != "" {
		return "schema.Formatted(" + strconv.Quote(shape.Format) + ")", 0, "string"
	}
	if known, precise := schema.PrecisionConstructors[shape.Precision]; precise {
		return known.Call, known.Carries, known.Holds
	}
	return kindExpression(shape.Kind), 0, kindType(shape.Kind)
}

// kindType is the Go type the unrefined constructor for a kind describes.
func kindType(kind structure.Kind) string {
	switch kind {
	case structure.Integer:
		return "int64"
	case structure.Number:
		return "float64"
	case structure.Boolean:
		return "bool"
	case structure.Bytes:
		return "[]byte"
	case structure.Timestamp:
		return "time.Time"
	default:
		return "string"
	}
}

func kindExpression(kind structure.Kind) string {
	switch kind {
	case structure.Integer:
		return "schema.Int64()"
	case structure.Number:
		return "schema.Float64()"
	case structure.Boolean:
		return "schema.Bool()"
	case structure.Bytes:
		return "schema.Bytes()"
	case structure.Timestamp:
		return "schema.Time()"
	default:
		return "schema.Text()"
	}
}

func sequenceExpression(shape structure.Sequence) (string, error) {
	listed, err := wrappedExpression("schema.List", shape.Element)
	if err != nil {
		return "", err
	}
	element, elementTypeed := elementType(shape.Element)
	if !elementTypeed && len(shape.Constraints) > 0 {
		return "", errors.New(
			"a bound on a list of that shape needs the element's Go type named, " +
				"and this description does not carry it")
	}
	return constrainExpression(listed, element, shape.Constraints)
}

// constrainExpression applies the constraints as one call, in the order the
// description recorded them.
//
// One call rather than nested ones, because that is the vocabulary now: a
// constraint is a value and the schema is what it is applied to, so the
// subject of a generated expression is on the left where a reader looks for
// it.
func constrainExpression(
	shape string,
	holds string,
	constraints []structure.Constraint,
) (string, error) {
	if len(constraints) == 0 {
		return shape, nil
	}
	applied := make([]string, 0, len(constraints))
	for _, constraint := range constraints {
		call, err := constraintCall(constraint, holds)
		if err != nil {
			return "", err
		}
		applied = append(applied, call)
	}
	return shape + ".Constrained(" + strings.Join(applied, ", ") + ")", nil
}

// constraintCall is one constraint as the call that builds it.
//
// A bound and an item count name the type they are about, because the value
// carries it and there is no longer an inner schema for the compiler to read
// it from. A length does not: it is only ever about text.
func constraintCall(constraint structure.Constraint, holds string) (string, error) {
	funced := func(call string, value string) string {
		return "schema." + call + "[" + holds + "](" + value + ")"
	}
	switch narrowed := constraint.(type) {
	case structure.AtLeast:
		return funced("AtLeast", bound(narrowed.Value)), nil
	case structure.AtMost:
		return funced("AtMost", bound(narrowed.Value)), nil
	case structure.Above:
		return funced("Above", bound(narrowed.Value)), nil
	case structure.Below:
		return funced("Below", bound(narrowed.Value)), nil
	case structure.MinLength:
		return "schema.MinLength(" + strconv.Itoa(narrowed.Value) + ")", nil
	case structure.MaxLength:
		return "schema.MaxLength(" + strconv.Itoa(narrowed.Value) + ")", nil
	case structure.Pattern:
		return "schema.Matching(" + strconv.Quote(narrowed.Expression) + ")", nil
	case structure.MinItems:
		return funced("MinItems", strconv.Itoa(narrowed.Value)), nil
	case structure.MaxItems:
		return funced("MaxItems", strconv.Itoa(narrowed.Value)), nil
	default:
		return "", errors.New("this constraint has no constructor")
	}
}

// elementType is the Go type a list's elements have, and whether the
// description says enough to name it.
//
// A scalar and a list of one it can name; an object, a union or a reference it
// cannot, because the Go type is the caller's own and the description carries
// only its schema name.
func elementType(node structure.Node) (string, bool) {
	switch shape := node.(type) {
	case structure.Scalar:
		_, _, holds := baseExpression(shape)
		return holds, true
	case structure.Sequence:
		within, elementTypeed := elementType(shape.Element)
		return "[]" + within, elementTypeed
	case structure.Nullable:
		within, elementTypeed := elementType(shape.Inner)
		return "*" + within, elementTypeed
	default:
		return "", false
	}
}

// bound renders a numeric bound as a Go literal. An untyped constant is used so
// the same text serves an int64 and a float64 shape.
func bound(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}

func wrappedExpression(combinator string, inner structure.Node) (string, error) {
	within, err := schemaExpression(inner)
	if err != nil {
		return "", err
	}
	return combinator + "(" + within + ")", nil
}

func namedExpression(name string, what string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("%s has no name, and a schema is referred to by name", what)
	}
	return name + "Schema", nil
}
