package schema

// How a struct's fields are described, indexed, written and read.
//
// Separate from the declaration surface because these are the mechanics: the
// order the fields are written in, the map a decoder looks a name up in, and
// what happens to a name nobody declared.

import (
	"strings"

	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

func describeFields[A any](fields []erasedField[A]) []structure.Field {
	descriptions := make([]structure.Field, 0, len(fields))
	for _, field := range fields {
		descriptions = append(descriptions, structure.Field{
			Name:        field.name,
			Exact:       field.literalName,
			Description: field.doc,
			Node:        field.node,
			Optional:    field.optional,
			Number:      field.number,
			Identity:    field.identity,
			Unique:      field.unique,
			UniqueKey:   field.uniqueKey,
			Computed:    field.computed,
			Default:     field.fallback,
		})
	}
	return descriptions
}

// firstFieldFault reports a duplicate name, an empty name, or a fault inherited
// from a field's own schema. Two fields with one name would make encoding and
// decoding disagree, so it is a declaration mistake and not a precedence rule.
func firstFieldFault[A any](fields []erasedField[A]) error {
	seen := make(map[string]bool, len(fields))
	for _, field := range fields {
		switch {
		case strings.TrimSpace(field.name) == "":
			return fail("a field has no name", nil)
		case seen[field.name]:
			return fail("two fields are named "+field.name, nil)
		case field.fault != nil:
			return within(field.name, field.fault)
		}
		seen[field.name] = true
	}
	return nil
}

func encodeFields[A any](value A, fields []erasedField[A], names []string, into Sink) error {
	if err := into.BeginObject(); err != nil {
		return err
	}
	for index, field := range fields {
		// An absent optional field is omitted rather than written as null.
		// Omission is what a reader of the projection is told to expect, and it
		// is what a document written by hand would do.
		if field.present != nil && !field.present(&value) {
			continue
		}
		if err := into.FieldName(names[index]); err != nil {
			return err
		}
		if err := field.encode(&value, into); err != nil {
			return within(names[index], err)
		}
	}
	return into.EndObject()
}

func decodeFields[A any](from Source, byName map[string]erasedField[A], required []string) (A, error) {
	var result A
	seen := make(map[string]bool, len(byName))

	err := from.ReadObject(func(name string) error {
		field, known := byName[name]
		if !known {
			// An unknown field is tolerated. A schema that rejected one could
			// not read a document written by a newer version of its producer.
			return from.Skip()
		}
		seen[name] = true
		return within(name, field.decode(&result, from))
	})
	if err != nil {
		var zero A
		return zero, err
	}

	for _, name := range required {
		if !seen[name] {
			var zero A
			return zero, within(name, fail("required field is missing", nil))
		}
	}
	return result, nil
}

// readValues reads an object's members as the values an Object's constructor
// is given, which is decodeFields with the fields' values kept apart rather
// than set into an A.
func readValues[A any](from Source, byName map[string]erasedField[A], required []string) (Values, error) {
	values := Values{entries: make(map[*fieldKey]slot, len(byName))}
	seen := make(map[string]bool, len(byName))

	err := from.ReadObject(func(name string) error {
		field, known := byName[name]
		if !known {
			return from.Skip()
		}
		seen[name] = true
		value, present, err := field.read(from)
		if err != nil {
			return within(name, err)
		}
		if present {
			values.entries[field.key] = value
		}
		return nil
	})
	if err != nil {
		return Values{}, err
	}
	for _, name := range required {
		if !seen[name] {
			return Values{}, within(name, fail("required field is missing", nil))
		}
	}
	return values, nil
}

// erase is an object's fields with their values' types put away.
func erase[A any](fields []ObjectField[A]) []erasedField[A] {
	erased := make([]erasedField[A], 0, len(fields))
	for _, field := range fields {
		erased = append(erased, field.erasure())
	}
	return erased
}
