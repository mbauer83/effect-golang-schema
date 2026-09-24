package schema

// What a field says beyond its name and shape: prose, a number, whether it is
// the identity, where its value comes from, and whether it may be absent.

import (
	"github.com/mbauer83/effect-golang-schema/schema/dynamic"
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// WithDescription attaches prose a projection can carry into its output.
func (field Field[A, B]) WithDescription(doc string) Field[A, B] {
	field.erased.doc = doc
	return field
}

// WithNumber gives the field a number, for a wire that identifies fields by
// number rather than by name.
//
// It is a modifier and not a parameter of FieldOf because most schemas never
// meet such a wire, and a number every declaration had to carry would be noise
// in all of them. Where one is needed it is required rather than derived: a
// number is what protobuf's compatibility rests on, so the description is where
// it belongs and declaration order is not a stable substitute.
func (field Field[A, B]) WithNumber(number int) Field[A, B] {
	if number < 1 {
		field.erased.fault = fail("a field number is at least 1", nil)
		return field
	}
	field.erased.number = number
	return field
}

// Identity marks the field that distinguishes one of these from another.
//
// A projection to storage makes it the key. A derived update shape leaves it
// out, because a key selects the row rather than being part of the row's new
// value. An object with one is an entity in its own right; an object without
// one is a value belonging to whatever holds it.
func (field Field[A, B]) Identity() Field[A, B] {
	field.erased.identity = true
	return field
}

// Unique marks a field no two values share: one account per email. It is a
// rule about all values rather than about one, which only storage can keep, so
// the domain states it and a mapping to storage enforces it.
func (field Field[A, B]) Unique() Field[A, B] {
	field.erased.unique = true
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
func (field Field[A, B]) Computed() Field[A, B] {
	field.erased.computed = true
	return field
}

// WithDefault says what the field holds when nobody gives it a value.
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
func (field Field[A, B]) WithDefault(value dynamic.Value) Field[A, B] {
	field.erased.fallback = structure.DefaultValue{Value: value}
	return field
}

// WithDefaultNow says the field holds the moment the row is written.
//
// Its own method rather than a value passed to WithDefault, because it is an
// expression and not a value: there is no instant to put in a description that
// would still be the right one when the row is written.
func (field Field[A, B]) WithDefaultNow() Field[A, B] {
	field.erased.fallback = structure.DefaultNow{}
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
func (field Field[A, B]) Optional() Field[A, B] {
	switch {
	case field.erased.present != nil:
		field.erased.optional = true
	case field.erased.derivable != nil:
		field.erased.present = field.erased.derivable
		field.erased.optional = true
	default:
		field.erased.fault = fail(
			"a bound field states presence in its getter; use OptionalFieldOf", nil)
		return field
	}
	field.erased = field.erased.tolerateAbsence()
	return field
}

// tolerateAbsence is what makes a field optional on the way *in* as well as
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
func (field erasedField[A]) tolerateAbsence() erasedField[A] {
	if inner := field.decode; inner != nil {
		field.decode = func(target *A, from Source) error {
			absent, err := from.Null()
			if err != nil || absent {
				return err
			}
			return inner(target, from)
		}
	}
	if inner := field.read; inner != nil {
		field.read = func(from Source) (slot, bool, error) {
			absent, err := from.Null()
			if err != nil || absent {
				return nil, false, err
			}
			return inner(from)
		}
	}
	return field
}
