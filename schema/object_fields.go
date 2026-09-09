package schema

import (
	"github.com/mbauer83/effect-golang-schema/schema/dynamic"
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// Field describes one member of A: its name on the wire, its shape, how to read
// it out of an A and how to write it into an A being built.
//
// The setter takes a pointer and the getter takes a value, so a struct is built
// from its zero value one field at a time. That is what avoids needing an
// N-argument constructor for every arity, and it is how a Go program builds a
// struct anyway.
type Field[A any] struct {
	name     string
	doc      string
	node     structure.Node
	optional bool
	number   int
	identity bool
	computed bool
	fallback structure.Default
	fault    error
	encode   func(A, Sink) error
	decode   func(*A, Source) error
	// present reports whether an optional field has a value to write. It is
	// nil for a required field, which always has one.
	present func(A) bool
	// derivable is the presence answer for a field whose absence can be seen
	// from the value itself, which is the case for a described field: the
	// member is in the object or it is not. Optional moves it into present.
	derivable func(A) bool
}

// FieldOf describes a required field.
func FieldOf[A, B any](
	name string,
	shape Schema[B],
	get func(A) B,
	set func(*A, B),
) Field[A] {
	return Field[A]{
		name:  name,
		node:  shape.node,
		fault: Validate(shape),
		encode: func(value A, into Sink) error {
			return Encode(shape, get(value), into)
		},
		decode: func(target *A, from Source) error {
			decoded, err := Decode(shape, from)
			if err != nil {
				return err
			}
			set(target, decoded)
			return nil
		},
	}
}

// OptionalFieldOf describes a field that may be absent.
//
// The getter reports whether the value is present, so absence is a decision the
// program makes rather than a zero value the schema has to guess about: an
// empty string that is meant to be sent is not the same as a field that is not
// there.
func OptionalFieldOf[A, B any](
	name string,
	shape Schema[B],
	get func(A) (B, bool),
	set func(*A, B),
) Field[A] {
	return Field[A]{
		name:     name,
		node:     shape.node,
		optional: true,
		fault:    Validate(shape),
		present: func(value A) bool {
			_, ok := get(value)
			return ok
		},
		encode: func(value A, into Sink) error {
			held, ok := get(value)
			if !ok {
				return into.Null()
			}
			return Encode(shape, held, into)
		},
		decode: func(target *A, from Source) error {
			absent, err := from.Null()
			if err != nil {
				return err
			}
			if absent {
				return nil
			}
			decoded, err := Decode(shape, from)
			if err != nil {
				return err
			}
			set(target, decoded)
			return nil
		},
	}
}

// Documented attaches prose a projection can carry into its output.
func (field Field[A]) Documented(doc string) Field[A] {
	field.doc = doc
	return field
}

// Numbered gives the field a number, for a wire that identifies fields by
// number rather than by name.
//
// It is a modifier and not a parameter of FieldOf because most schemas never
// meet such a wire, and a number every declaration had to carry would be noise
// in all of them. Where one is needed it is required rather than derived: a
// number is what protobuf's compatibility rests on, so the description is where
// it belongs and declaration order is not a stable substitute.
func (field Field[A]) Numbered(number int) Field[A] {
	if number < 1 {
		field.fault = fail("a field number is at least 1", nil)
		return field
	}
	field.number = number
	return field
}

// Identity marks the field that distinguishes one of these from another.
//
// A projection to storage makes it the key. A derived update shape leaves it
// out, because a key selects the row rather than being part of the row's new
// value. An object with one is an entity in its own right; an object without
// one is a value belonging to whatever holds it.
func (field Field[A]) Identity() Field[A] {
	field.identity = true
	return field
}

// Computed marks a field whose value comes from somewhere other than the
// caller: a default, a trigger, a derivation.
//
// It is left out of every derived shape a caller supplies, because asking for a
// value that will be overwritten is asking a question with no answer.
//
// Compose it with Identity for a key the database generates, and use Identity
// alone for one the application generates. That composition is the distinction
// other libraries spell with two separate concepts.
func (field Field[A]) Computed() Field[A] {
	field.computed = true
	return field
}

// Defaulting says what the field holds when nobody gives it a value.
//
// The value is a dynamic.Value rather than a Go value because a description
// need not have a Go type at all, and because every projection already knows
// how to write one.
//
// It is separate from Computed, which says the value is not the caller's. A
// field can have a default and still be the caller's to give -- that is what a
// default *is* -- and a computed field with no default is a projection to
// storage's problem rather than a declaration mistake, so the two are declared
// separately and each says its own thing.
func (field Field[A]) Defaulting(value dynamic.Value) Field[A] {
	field.fallback = structure.DefaultTo{Value: value}
	return field
}

// DefaultingToNow says the field holds the moment the row is written.
//
// Its own method rather than a value passed to Defaulting, because it is an
// expression and not a value: there is no instant to put in a description that
// would still be the right one when the row is written.
func (field Field[A]) DefaultingToNow() Field[A] {
	field.fallback = structure.DefaultNow{}
	return field
}

// Optional marks a field that may be absent.
//
// It applies to a field whose presence is answerable: one describing a shape,
// where absence is the member not being there, and one already built by
// OptionalFieldOf. A field bound to a Go type with FieldOf cannot be made
// optional this way, because its getter returns a value and not a value and
// whether there is one -- and absence is a decision the program makes rather
// than a zero value the schema guesses at. That is why OptionalFieldOf takes a
// different getter rather than this taking none.
func (field Field[A]) Optional() Field[A] {
	switch {
	case field.present != nil:
		field.optional = true
	case field.derivable != nil:
		field.present = field.derivable
		field.optional = true
	default:
		field.fault = fail(
			"a bound field states presence in its getter; use OptionalFieldOf", nil)
		return field
	}
	return field.tolerantOfAbsence()
}

// tolerantOfAbsence is what makes a field optional on the way *in* as well as
// out.
//
// Saying a field is optional used to change only the description and the
// encoding: a decoder still demanded a value, so a source that answered with an
// explicit absence -- a JSON null, a SQL NULL -- was refused by the field that
// had just declared it could be missing. Optionality that holds in one
// direction is not optionality, and the shape of the bug was that a projection
// to storage said the column was nullable while the description could not read
// one back.
//
// OptionalFieldOf does not come through here: it states absence in its getter
// and consults Null in its own decoder, so wrapping it again would consume the
// absence twice.
func (field Field[A]) tolerantOfAbsence() Field[A] {
	inner := field.decode
	if inner == nil {
		return field
	}
	field.decode = func(target *A, from Source) error {
		absent, err := from.Null()
		if err != nil {
			return err
		}
		if absent {
			return nil
		}
		return inner(target, from)
	}
	return field
}
