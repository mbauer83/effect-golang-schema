package schema

import (
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// Field describes one member of A whose value is a B: its name on the wire,
// its shape, and how to read it out of an A.
//
// An A is made from its fields in one of two ways. Struct builds it from its
// zero value, one setter at a time, which is how a Go program fills a plain
// struct. Object hands the decoded values to a constructor, which reads each
// through its field -- Of, Lookup -- and can refuse them, which is how a type
// with rules is made: through the one function that enforces them.
//
// A field is also a handle. It keeps its identity through every modifier, so
// the value a constructor reads with it is the value it decoded, and a mapping
// or a query elsewhere can name it by the Go value rather than by a string that
// a rename would leave behind.
type Field[A, B any] struct {
	erased erasedField[A]
	// lookup reads the value out of an A, and whether it has one; set writes
	// it into an A being built, and is nil for a field declared without a
	// setter. They are kept typed so a projection can represent the field
	// another way.
	lookup func(A) (B, bool)
	set    func(*A, B)
}

// ObjectField is a field of A, whatever the type of its value: what Struct
// and Object take, since an object's fields hold different types.
type ObjectField[A any] interface {
	erasure() erasedField[A]
}

func (field Field[A, B]) erasure() erasedField[A] { return field.erased }

// erasedField is a field with its value's type put away: what an object's
// codec needs to write, read and describe it.
type erasedField[A any] struct {
	name     string
	doc      string
	node     structure.Node
	optional bool
	number   int
	identity bool
	computed bool
	fallback structure.Default
	fault    error
	// key is the field's identity, shared by every copy a modifier makes.
	key *fieldKey
	// literalName says the name was given exactly, by a projection, and is
	// not respelled by a format's naming strategy.
	literalName bool
	encode      func(A, Sink) error
	// decode writes the field into an A being built. It is nil for a field
	// declared without a setter, which only Object can use.
	decode func(*A, Source) error
	// read decodes the field's value alone, and whether there is one, for
	// Object to hand to a constructor.
	read func(Source) (slot, bool, error)
	// present reports whether an optional field has a value to write. It is
	// nil for a required field, which always has one.
	present func(A) bool
	// derivable is the presence answer for a field whose absence can be seen
	// from the value itself, which is the case for a described field: the
	// member is in the object or it is not. Optional moves it into present.
	derivable func(A) bool
}

// fieldKey is what makes two copies of a field the same field. It has a
// member so that two keys are two allocations: Go may give every allocation of
// a zero-sized type the same address.
type fieldKey struct{ name string }

// FieldOf describes a required field.
//
// The setter is for Struct, which builds an A by setting its fields, and is
// left out for Object, which builds one through a constructor. It is variadic
// only so that it can be left out: more than one setter is refused.
func FieldOf[A, B any](
	name string,
	shape Schema[B],
	get func(A) B,
	set ...func(*A, B),
) Field[A, B] {
	field := erasedField[A]{
		name:  name,
		node:  shape.node,
		fault: Validate(shape),
		key:   &fieldKey{name: name},
		encode: func(value A, into Sink) error {
			return Encode(shape, get(value), into)
		},
		read: func(from Source) (slot, bool, error) {
			value, err := Decode(shape, from)
			return typedSlot[B]{value: value}, true, err
		},
	}
	switch len(set) {
	case 0:
	case 1:
		assign := set[0]
		field.decode = func(target *A, from Source) error {
			value, err := Decode(shape, from)
			if err != nil {
				return err
			}
			assign(target, value)
			return nil
		}
	default:
		field.fault = fail("a field has at most one setter", nil)
	}
	return Field[A, B]{
		erased: field,
		lookup: func(value A) (B, bool) { return get(value), true },
		set:    firstSetter(set),
	}
}

// OptionalFieldOf describes a field that may be absent.
//
// The getter reports whether the value is present, so absence is a decision the
// program makes rather than a zero value the schema has to guess about: an
// empty string that is meant to be sent is not the same as a field that is not
// there. The setter is optional for the reason it is in FieldOf.
func OptionalFieldOf[A, B any](
	name string,
	shape Schema[B],
	get func(A) (B, bool),
	set ...func(*A, B),
) Field[A, B] {
	field := erasedField[A]{
		name:     name,
		node:     shape.node,
		optional: true,
		fault:    Validate(shape),
		key:      &fieldKey{name: name},
		present: func(value A) bool {
			_, ok := get(value)
			return ok
		},
		encode: func(value A, into Sink) error {
			member, ok := get(value)
			if !ok {
				return into.Null()
			}
			return Encode(shape, member, into)
		},
		read: func(from Source) (slot, bool, error) {
			absent, err := from.Null()
			if err != nil || absent {
				return nil, false, err
			}
			value, err := Decode(shape, from)
			return typedSlot[B]{value: value}, true, err
		},
	}
	switch len(set) {
	case 0:
	case 1:
		assign := set[0]
		field.decode = func(target *A, from Source) error {
			absent, err := from.Null()
			if err != nil || absent {
				return err
			}
			value, err := Decode(shape, from)
			if err != nil {
				return err
			}
			assign(target, value)
			return nil
		}
	default:
		field.fault = fail("a field has at most one setter", nil)
	}
	return Field[A, B]{erased: field, lookup: get, set: firstSetter(set)}
}

func firstSetter[A, B any](set []func(*A, B)) func(*A, B) {
	if len(set) == 0 {
		return nil
	}
	return set[0]
}
