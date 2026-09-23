package schemagen

// Binding a described union to Go: an interface with an unexported marker, and
// the schema that narrows to each variant. That is how Go spells a sum, and the
// marker being unexported is what keeps the set closed.

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

func writeUnionBinding(out *bytes.Buffer, shape structure.Union) error {
	if len(shape.Variants) == 0 {
		return fmt.Errorf("%s has no variants", shape.Name)
	}
	marker := markerName(shape.Name)

	writeDoc(out, shape.Name, shape.Doc)
	fmt.Fprintf(out, "type %s interface{ %s() }\n", shape.Name, marker)

	fmt.Fprintf(out, "\n// %sSchema describes %s. It is generated from its description.\n",
		shape.Name, shape.Name)
	fmt.Fprintf(out, "var %sSchema = ", shape.Name)
	fmt.Fprintf(out, "schema.OneOf[%s](%s,\n", shape.Name, strconv.Quote(shape.Name))
	for _, variant := range shape.Variants {
		if err := writeVariant(out, shape.Name, variant); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, ")")
	if shape.Doc != "" {
		fmt.Fprintf(out, ".WithDescription(%s)", strconv.Quote(shape.Doc))
	}
	fmt.Fprintf(out, "\n")
	return nil
}

func writeVariant(out *bytes.Buffer, union string, variant structure.Variant) error {
	object, isObject := variant.Node.(structure.Object)
	if !isObject {
		return fmt.Errorf("%s.%s: a variant becomes a Go type, so it is an object",
			union, variant.Name)
	}
	fmt.Fprintf(out, "schema.VariantOf(%s, %sSchema,\n",
		strconv.Quote(variant.Name), object.Name)
	fmt.Fprintf(out, "func(value %s) (%s, bool) { variant, is := value.(%s); return variant, is },\n",
		union, object.Name, object.Name)
	fmt.Fprintf(out, "func(variant %s) %s { return variant })", object.Name, union)
	if variant.Doc != "" {
		fmt.Fprintf(out, ".WithDescription(%s)", strconv.Quote(variant.Doc))
	}
	fmt.Fprintf(out, ",\n")
	return nil
}
