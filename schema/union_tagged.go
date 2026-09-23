package schema

// A union whose variants are told apart by a field inside them.
//
//	{"type": "iso2768", "grade": "medium"}
//
// This is the common REST shape and it costs more than naming the variant as
// the object's key. The name may arrive after the fields whose meaning it
// settles, so a decoder has to read the object before it knows what it read --
// which is why it asks the format for that power rather than assuming the
// ordering, and why the other form is still the one to reach for when nothing
// external dictates the wire.

import (
	"strings"

	"github.com/mbauer83/effect-golang-schema/schema/dynamic"
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// OneOfBy describes A as a choice between variants told apart by a field.
//
// The field is written by the union and is not part of a variant's own shape,
// so a Go type does not carry a tag it never reads. A variant that declares a
// field of the same name is a declaration mistake: one of the two would win and
// which is not something to leave to chance.
//
// Every variant must be an object, because a field inside it is where the name
// goes. Variants are tried in declared order on encode, as OneOf's are.
func OneOfBy[A any](name string, discriminator string, variants ...Variant[A]) Schema[A] {
	node := structure.Union{
		Name:          name,
		Discriminator: discriminator,
		Variants:      describeVariants(variants),
	}
	if fault := firstTaggedFault(discriminator, variants); fault != nil {
		return faultySchema[A](node, fault)
	}

	byName := make(map[string]Variant[A], len(variants))
	for _, variant := range variants {
		byName[variant.name] = variant
	}
	return of[A](
		node,
		func(value A, into Sink) error {
			return encodeTagged(value, discriminator, variants, into)
		},
		func(from Source) (A, error) {
			return decodeTagged[A](from, discriminator, byName)
		},
	)
}

// firstTaggedFault reports what would make the union unusable, at the moment it
// is declared.
func firstTaggedFault[A any](discriminator string, variants []Variant[A]) error {
	if strings.TrimSpace(discriminator) == "" {
		return fail("a union told apart by a field needs the field's name", nil)
	}
	if fault := firstVariantFault(variants); fault != nil {
		return fault
	}
	for _, variant := range variants {
		object, isObject := variant.node.(structure.Object)
		if !isObject {
			return within(variant.name,
				fail("is not an object, and the name goes in a field of one", nil))
		}
		for _, field := range object.Fields {
			if field.Name == discriminator {
				return within(variant.name,
					fail("already has a field named "+discriminator, nil))
			}
		}
	}
	return nil
}

func encodeTagged[A any](
	value A,
	discriminator string,
	variants []Variant[A],
	into Sink,
) error {
	for _, variant := range variants {
		if !variant.matches(value) {
			continue
		}
		// Every variant is an object -- the declaration check above refuses
		// anything else -- so the name always has somewhere to go.
		sink := &tagSink{Sink: into, name: discriminator, value: variant.name}
		if err := variant.encode(value, sink); err != nil {
			return within(variant.name, err)
		}
		return nil
	}
	return fail("no variant matches the value", nil)
}

// tagSink writes the name into the object the variant writes, which is what
// puts the two in one object without the variant knowing about the union.
type tagSink struct {
	Sink
	name  string
	value string
	depth int
}

func (sink *tagSink) BeginObject() error {
	if err := sink.Sink.BeginObject(); err != nil {
		return err
	}
	sink.depth++
	if sink.depth > 1 {
		return nil
	}
	if err := sink.Sink.FieldName(sink.name); err != nil {
		return err
	}
	return sink.Sink.Text(sink.value)
}

func (sink *tagSink) EndObject() error {
	sink.depth--
	return sink.Sink.EndObject()
}

// decodeTagged reads the object, finds the name, and then reads it again as the
// variant that name selects.
func decodeTagged[A any](
	from Source,
	discriminator string,
	byName map[string]Variant[A],
) (A, error) {
	var zero A
	bufferer, can := from.(Bufferer)
	if !can {
		return zero, fail(
			"this format cannot read a union told apart by a field, because the "+
				"name may arrive after the fields it settles", nil)
	}
	document, err := bufferer.Buffer()
	if err != nil {
		return zero, err
	}

	object, isObject := document.(dynamic.Object)
	if !isObject {
		return zero, fail("is not an object", nil)
	}
	tag, err := tagOf(object, discriminator)
	if err != nil {
		return zero, err
	}
	variant, known := byName[tag]
	if !known {
		return zero, fail("no variant is named "+tag, nil)
	}
	// The name is the union's, not the variant's, so the variant reads the
	// object it would have written: its own fields and nothing else.
	value, err := variant.decode(&dynamicSource{
		queue: []dynamic.Value{omit(object, discriminator)},
	})
	if err != nil {
		return zero, within(tag, err)
	}
	return value, nil
}

func tagOf(object dynamic.Object, discriminator string) (string, error) {
	member, present := object.Member(discriminator)
	if !present {
		return "", fail("carries no "+discriminator+" to say which variant it is", nil)
	}
	text, isText := member.(dynamic.Text)
	if !isText {
		return "", within(discriminator, fail("is not text", nil))
	}
	return text.Value, nil
}

func omit(object dynamic.Object, name string) dynamic.Value {
	rest := dynamic.Object{Fields: make([]dynamic.Field, 0, len(object.Fields))}
	for _, field := range object.Fields {
		if field.Name != name {
			rest.Fields = append(rest.Fields, field)
		}
	}
	return rest
}
