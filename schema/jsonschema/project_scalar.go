package jsonschema

// A scalar and what narrows it: the type keyword, the format, and the bounds.

import "github.com/mbauer83/effect-golang-schema/schema/structure"

func scalarNode(shape structure.Scalar) Node {
	schema := Node{
		Format: formatFor(shape),
		Bounds: bounds(shape.Constraints),
	}
	switch shape.Kind {
	case structure.Integer:
		schema.Type = "integer"
	case structure.Number:
		schema.Type = "number"
	case structure.Boolean:
		schema.Type = "boolean"
		schema.Format = ""
	default:
		// Text, Bytes and Timestamp are all strings on the wire; their format
		// is what tells them apart, and the schema already set it.
		schema.Type = "string"
	}
	return schema
}

// formatFor is the format keyword the shape carries, or the one its Go width
// implies.
//
// Only the widths the specification registers a name for are emitted. The
// others are stated by minimum and maximum, which is what a reader can act on;
// putting "uint16" in a contract would be telling a client about Go.
func formatFor(shape structure.Scalar) string {
	if shape.Format != "" {
		return shape.Format
	}
	return precisionFormats[shape.Precision]
}

var precisionFormats = map[structure.Precision]string{
	structure.Int32Bits:   "int32",
	structure.Int64Bits:   "int64",
	structure.Float32Bits: "float",
	structure.Float64Bits: "double",
}

// bounds translates the constraint vocabulary into the keywords that say the
// same thing. A constraint with no keyword would be silently dropped, so the
// switch is exhaustive over a sealed set for exactly that reason.
func bounds(constraints []structure.Constraint) Bounds {
	keywords := Bounds{}
	for _, constraint := range constraints {
		switch constraint := constraint.(type) {
		case structure.AtLeast:
			keywords.Minimum = &constraint.Value
		case structure.AtMost:
			keywords.Maximum = &constraint.Value
		case structure.Above:
			keywords.ExclusiveMinimum = &constraint.Value
		case structure.Below:
			keywords.ExclusiveMaximum = &constraint.Value
		case structure.MinLength:
			keywords.MinLength = &constraint.Value
		case structure.MaxLength:
			keywords.MaxLength = &constraint.Value
		case structure.Pattern:
			keywords.Pattern = constraint.Expression
		case structure.MinItems:
			keywords.MinItems = &constraint.Value
		case structure.MaxItems:
			keywords.MaxItems = &constraint.Value
		}
	}
	return keywords
}
