package schema

// An object as a whole: the fixed set of named fields a struct is, and how one
// is written and read.
//
// The fields themselves are in object_fields.go; this is the shape they add up
// to.

import (
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// Struct describes A as a fixed set of named fields, built from A's zero value
// by setting each field in turn.
//
// It is for a plain struct, whose fields are its whole story. A type with rules
// -- unexported fields, a constructor that refuses -- is described with Object,
// which builds it through that constructor instead. Every field of a Struct
// needs a setter.
//
// A named struct becomes a reusable component in projections that have them, so
// a type used by ten endpoints is described once. Pass an empty name to inline
// it instead.
func Struct[A any](name string, fields ...ObjectField[A]) Schema[A] {
	return assemble(structure.Object{Name: name}, &objectParts[A]{fields: erase(fields)})
}

// Object describes A as a fixed set of named fields, built by construct from
// the values they decode.
//
// The constructor reads each value through the field that declared it -- Of,
// Lookup -- and may refuse them, so a value that decodes is one construct would
// have made: its rules hold whether it came from a caller, a request or a row.
// Encoding reads each field through its getter, so A needs to export nothing
// beyond what it already does.
//
// A constructor that reads a field the object does not declare is refused
// here, when the schema is built, rather than silently reading a zero value on
// every decode.
func Object[A any](name string, construct func(Values) (A, error), fields ...ObjectField[A]) Schema[A] {
	parts := &objectParts[A]{fields: erase(fields), construct: construct}
	if fault := probeConstructor(construct, parts.fields); fault != nil {
		built := faultySchema[A](structure.Object{Name: name, Fields: describeFields(parts.fields)}, fault)
		built.object = parts
		return built
	}
	return assemble(structure.Object{Name: name}, parts)
}

// objectParts are what an object schema is built from, kept so that a
// projection of it can be built from them too.
type objectParts[A any] struct {
	fields []erasedField[A]
	// construct builds an A from decoded values, for an Object. It is nil for
	// a Struct, which sets its fields instead.
	construct func(Values) (A, error)
	// omitted says a projection left fields out. An Object reads a document
	// written with it back through its constructor, which is given the omitted
	// members as absent and decides; a Struct cannot, since nothing would
	// decide and the members would silently be zero.
	omitted bool
}

// assemble is the schema an object's parts make, under node's name and
// description.
func assemble[A any](node structure.Object, parts *objectParts[A]) Schema[A] {
	node.Fields = describeFields(parts.fields)
	faulty := func(fault error) Schema[A] {
		built := faultySchema[A](node, fault)
		built.object = parts
		return built
	}
	if fault := firstFieldFault(parts.fields); fault != nil {
		return faulty(fault)
	}
	if parts.construct == nil && !parts.omitted {
		for _, field := range parts.fields {
			if field.decode == nil {
				return faulty(within(field.name, fail(
					"a Struct sets its fields, and this one has no setter; "+
						"describe a type built by a constructor with Object", nil)))
			}
		}
	}

	names := newMemberNames(parts.fields)
	built := of[A](
		node,
		func(value A, into Sink) error {
			spelling := names.under(sinkStrategy(into))
			if spelling.fault != nil {
				return spelling.fault
			}
			return encodeFields(value, parts.fields, spelling.names, into)
		},
		func(from Source) (A, error) {
			var zero A
			spelling := names.under(sourceStrategy(from))
			switch {
			case spelling.fault != nil:
				return zero, spelling.fault
			case parts.omitted && parts.construct == nil:
				return zero, fail("this projection of a Struct leaves members out, and "+
					"nothing would decide what they hold; read what it writes as a "+
					"description, with Dynamic, or describe the type with Object", nil)
			case parts.construct == nil:
				return decodeFields[A](from, spelling.byName, spelling.required)
			}
			values, err := readValues(from, spelling.byName, spelling.required)
			if err != nil {
				return zero, err
			}
			value, err := parts.construct(values)
			if err != nil {
				return zero, refusalError(err)
			}
			return value, nil
		},
	)
	built.object = parts
	return built
}
