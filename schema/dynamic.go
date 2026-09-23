package schema

// A schema for a shape with no Go type.
//
// Schema[A] moves a Go value in and out of a format. A description written
// before the type it will become exists -- or loaded from somewhere else, or
// read back out of another schema -- has no A, and it is still worth
// validating, transcoding, inspecting and composing. The universal
// representation is the A it uses instead.
//
// There is no second codec. Dynamic rebuilds the description out of the same
// combinators a typed schema is written with, so the two paths cannot disagree
// about what a shape admits: they are the same path. That is also why every
// rule the description records is enforced without anything here knowing what
// the rules are.

import (
	"github.com/mbauer83/effect-golang-schema/schema/dynamic"
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// Dynamic derives a schema over the universal representation from a
// description.
//
// It is the door between the two ways of using this package: a typed schema's
// Structure passed through here is usable without its type, and a description
// assembled by hand or loaded from elsewhere is usable at all.
//
// What it enforces is what the description records. A bound, a length, a
// pattern, an item count, a width and the shape itself are all there; a rule
// that could only be code -- an address, a URI, a refinement -- annotates and
// no more, because there was nothing to record. That is the honest limit of
// describing a shape rather than writing one.
func Dynamic(node structure.Node) Schema[dynamic.Value] {
	codec := dynamicCodec(node)
	if fault := Validate(codec); fault != nil {
		return faultySchema[dynamic.Value](node, fault)
	}
	// The description is kept exactly as it was given. Rebuilding it produced a
	// codec, not a new description, and a projection has to see what the author
	// wrote -- the recorded width included, which the rebuilt wire shape does
	// not carry.
	return of(node, codec.encode, codec.decode)
}

func dynamicCodec(node structure.Node) Schema[dynamic.Value] {
	switch shape := node.(type) {
	case structure.Scalar:
		return dynamicScalar(shape)
	case structure.Object:
		return dynamicObject(shape)
	case structure.Sequence:
		return dynamicSequence(shape)
	case structure.Mapping:
		return dynamicMapping(shape)
	case structure.Union:
		return dynamicUnion(shape)
	case structure.Nullable:
		return dynamicNullable(shape)
	case structure.Reference:
		return dynamicReference(shape)
	default:
		return faultySchema[dynamic.Value](node, fail("a description is required", nil))
	}
}

// DynamicField declares a field of a description: a name and a shape, and no
// accessors.
//
// The accessors are the only part of a field declaration that needs a Go type,
// so leaving them out is exactly the difference between describing a shape and
// binding one. Everything else is the same Field, so the same modifiers apply:
// Optional and WithDescription.
//
// It is named for what it returns, as FieldOf is. A third word for a field --
// a member, an entry -- would be one more thing to learn about something the
// reader already knows.
func DynamicField[B any](name string, shape Schema[B]) Field[dynamic.Value] {
	codec := Dynamic(shape.Structure())
	if fault := Validate(shape); fault != nil {
		return Field[dynamic.Value]{name: name, node: shape.Structure(), fault: fault}
	}
	return Field[dynamic.Value]{
		name:      name,
		node:      shape.Structure(),
		fault:     Validate(codec),
		derivable: func(value dynamic.Value) bool { return hasMember(value, name) },
		encode: func(value dynamic.Value, into Sink) error {
			member, present := memberOf(value, name)
			if !present {
				return fail("required member is missing", nil)
			}
			return Encode(codec, member, into)
		},
		decode: func(target *dynamic.Value, from Source) error {
			value, err := Decode(codec, from)
			if err != nil {
				return err
			}
			*target = withMember(*target, name, value)
			return nil
		},
	}
}

// DynamicVariant declares a variant of a described union: a name and a
// shape, and no narrowing.
//
// A described value carries its own tag -- the chosen variant is the object's
// single member -- so narrowing to it is reading that name, which needs no Go
// type either.
//
// It is a variant and not an alternative or a choice, because the type it
// returns is Variant and OneOf takes Variants. The word was already chosen.
func DynamicVariant[B any](name string, shape Schema[B]) Variant[dynamic.Value] {
	codec := Dynamic(shape.Structure())
	if fault := Validate(shape); fault != nil {
		return Variant[dynamic.Value]{name: name, node: shape.Structure(), fault: fault}
	}
	return Variant[dynamic.Value]{
		name:  name,
		node:  shape.Structure(),
		fault: Validate(codec),
		matches: func(value dynamic.Value) bool {
			variant, only := variantOf(value)
			return only && variant.Name == name
		},
		encode: func(value dynamic.Value, into Sink) error {
			variant, only := variantOf(value)
			if !only || variant.Name != name {
				return fail("the value is not the "+name+" variant", nil)
			}
			return Encode(codec, variant.Value, into)
		},
		decode: func(from Source) (dynamic.Value, error) {
			value, err := Decode(codec, from)
			if err != nil {
				return nil, err
			}
			return dynamic.Object{Fields: []dynamic.Field{{Name: name, Value: value}}}, nil
		},
	}
}

// memberOf reads a member of an object, which is where a described field's
// value lives.
func memberOf(value dynamic.Value, name string) (dynamic.Value, bool) {
	object, isObject := value.(dynamic.Object)
	if !isObject {
		return nil, false
	}
	return object.Member(name)
}

func hasMember(value dynamic.Value, name string) bool {
	_, present := memberOf(value, name)
	return present
}

// withMember adds a member to the object being built, which starts as nothing
// because a decoder builds from a zero value.
func withMember(target dynamic.Value, name string, value dynamic.Value) dynamic.Value {
	object, isObject := target.(dynamic.Object)
	if !isObject {
		object = dynamic.Object{}
	}
	object.Fields = append(object.Fields, dynamic.Field{Name: name, Value: value})
	return object
}

func variantOf(value dynamic.Value) (dynamic.Field, bool) {
	object, isObject := value.(dynamic.Object)
	if !isObject {
		return dynamic.Field{}, false
	}
	return object.Only()
}
