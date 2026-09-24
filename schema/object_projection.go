package schema

// A projection of an object: the same object with members left out, named
// exactly, or represented another way. It is how a surface publishes a domain
// type under its own terms -- what is sent, what each member is called, what
// form a value takes -- without a second struct of the same members.

import (
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// Omit is this object without fields, which are then neither written nor
// expected.
//
// Reading a document written with it back depends on how A is made. An
// Object's constructor is given the omitted members as absent -- Lookup
// reports them missing -- and decides whether that makes an A, so its rules
// hold either way. A Struct refuses, because nothing would decide and the
// members would silently be zero; a reader that wants such a document reads it
// as a description, with Dynamic.
func (schema Schema[A]) Omit(fields ...ObjectField[A]) Schema[A] {
	parts, node, fault := schema.projectable()
	if fault != nil {
		return schema.withFault(fault)
	}
	leaving := make(map[*fieldKey]bool, len(fields))
	for _, field := range fields {
		key := field.erasure().key
		if parts.index(key) < 0 {
			return schema.withFault(fail(field.erasure().name+" is not a field of this object", nil))
		}
		leaving[key] = true
	}
	kept := make([]erasedField[A], 0, len(parts.fields))
	for _, field := range parts.fields {
		if !leaving[field.key] {
			kept = append(kept, field)
		}
	}
	return assemble(node, &objectParts[A]{fields: kept, construct: parts.construct, omitted: true})
}

// Rename gives a field a name of its own in this projection, used exactly as
// written: no format's naming strategy respells it.
func (schema Schema[A]) Rename(field ObjectField[A], name string) Schema[A] {
	return schema.replacing(field.erasure(), func(current erasedField[A]) erasedField[A] {
		current.name = name
		current.literalName = true
		return current
	})
}

// Describe gives a field this projection's own prose, for a member whose
// meaning here is not the domain's: a path the domain keeps, published as a
// URL, is described as the URL it is.
func (schema Schema[A]) Describe(field ObjectField[A], doc string) Schema[A] {
	return schema.replacing(field.erasure(), func(current erasedField[A]) erasedField[A] {
		current.doc = doc
		return current
	})
}

// Represent is this object with field written and read as a C: to converts
// its value on the way out, and from converts it back on the way in, so the
// value still reaches the constructor or the setter as the B it is.
//
// A path the domain keeps, published as a URL; a duration kept as one, written
// as a count of minutes: the member is the same, and so is its name, but not
// its form.
func (schema Schema[A]) Represent[B, C any](field Field[A, B], shape Schema[C], to func(B) C, from func(C) B) Schema[A] {
	return schema.replacing(field.erased, func(current erasedField[A]) erasedField[A] {
		lookup, set, optional := field.lookup, field.set, current.optional
		readC := func(source Source) (C, bool, error) {
			var zero C
			if optional {
				absent, err := source.Null()
				if err != nil || absent {
					return zero, false, err
				}
			}
			value, err := Decode(shape, source)
			return value, err == nil, err
		}
		current.node = shape.node
		if fault := Validate(shape); fault != nil {
			current.fault = fault
		}
		current.encode = func(value A, into Sink) error {
			member, present := lookup(value)
			if !present {
				return into.Null()
			}
			return Encode(shape, to(member), into)
		}
		current.read = func(source Source) (slot, bool, error) {
			value, present, err := readC(source)
			if err != nil || !present {
				return nil, false, err
			}
			return typedSlot[B]{value: from(value)}, true, nil
		}
		current.decode = nil
		if set != nil {
			current.decode = func(target *A, source Source) error {
				value, present, err := readC(source)
				if err == nil && present {
					set(target, from(value))
				}
				return err
			}
		}
		return current
	})
}

// Reshape is this object with field written and read by another schema of
// the same type: a nested object's own projection, most often.
func (schema Schema[A]) Reshape[B any](field Field[A, B], shape Schema[B]) Schema[A] {
	same := func(value B) B { return value }
	return schema.Represent(field, shape, same, same)
}

func (schema Schema[A]) replacing(field erasedField[A], change func(erasedField[A]) erasedField[A]) Schema[A] {
	parts, node, fault := schema.projectable()
	if fault != nil {
		return schema.withFault(fault)
	}
	index := parts.index(field.key)
	if index < 0 {
		return schema.withFault(fail(field.name+" is not a field of this object", nil))
	}
	fields := append([]erasedField[A](nil), parts.fields...)
	fields[index] = change(fields[index])
	return assemble(node, &objectParts[A]{fields: fields, construct: parts.construct, omitted: parts.omitted})
}

// projectable is the object this schema describes, or why it has none to
// project.
func (schema Schema[A]) projectable() (*objectParts[A], structure.Object, error) {
	if fault := Validate(schema); fault != nil {
		return nil, structure.Object{}, fault
	}
	node, isObject := schema.node.(structure.Object)
	if !isObject || schema.object == nil {
		return nil, structure.Object{}, fail("only an object built by Struct or Object can be projected", nil)
	}
	return schema.object, node, nil
}

func (schema Schema[A]) withFault(fault error) Schema[A] {
	return faultySchema[A](schema.node, fault)
}

func (parts *objectParts[A]) index(key *fieldKey) int {
	for index, field := range parts.fields {
		if field.key == key {
			return index
		}
	}
	return -1
}
