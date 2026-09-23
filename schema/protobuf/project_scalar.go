package protobuf

// The proto3 type a scalar is, and the notes proto3 has no way to state.
//
// A Kind says what the value is on the wire and a Precision says what it is in
// a program. Protobuf has a type for each width, so this is the one projection
// where the Precision is not detail: int32 and int64 are different types on the
// wire, and choosing the wider one for a schema that said int32 would change
// what every other language generates.

import (
	"strconv"

	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

func (projection *projector) scalar(scalar structure.Scalar) (shape, error) {
	notes := constraintNotes(scalar.Constraints)
	if scalar.Format != "" {
		notes = append(notes, "format: "+scalar.Format)
	}

	switch scalar.Kind {
	case structure.Text:
		return shape{name: "string", notes: notes}, nil
	case structure.Boolean:
		return shape{name: "bool", notes: notes}, nil
	case structure.Bytes:
		return shape{name: "bytes", notes: notes}, nil
	case structure.Timestamp:
		projection.addImport(TimestampImport)
		return shape{name: "google.protobuf.Timestamp", notes: notes}, nil
	case structure.Number:
		if scalar.Precision == structure.Float32Bits {
			return shape{name: "float", notes: notes}, nil
		}
		return shape{name: "double", notes: notes}, nil
	default:
		return shape{name: integerType(scalar.Precision), notes: notes}, nil
	}
}

// integerType is the proto3 integer a precision is.
//
// The narrow widths widen to the narrowest proto3 type that holds them, which
// is int32 for the signed ones and uint32 for the unsigned: proto3 has no int8,
// and a schema that said int8 still means a number that fits in one. The range
// the schema stated is carried as a note, since proto3 cannot state it.
func integerType(precision structure.Precision) string {
	switch precision {
	case structure.Int8Bits, structure.Int16Bits, structure.Int32Bits:
		return "int32"
	case structure.Uint8Bits, structure.Uint16Bits, structure.Uint32Bits:
		return "uint32"
	case structure.Uint64Bits, structure.UintBits:
		return "uint64"
	default:
		return "int64"
	}
}

// constraintNotes is what the constraints say, as prose.
//
// Proto3 has no validation keywords. A comment is honest about not being
// enforced, where an invented option would look like a rule the wire carried --
// and the server does enforce them, through the same schema.
func constraintNotes(constraints []structure.Constraint) []string {
	if len(constraints) == 0 {
		return nil
	}
	notes := make([]string, 0, len(constraints))
	for _, constraint := range constraints {
		if note := constraintText(constraint); note != "" {
			notes = append(notes, note)
		}
	}
	return notes
}

func constraintText(constraint structure.Constraint) string {
	switch inner := constraint.(type) {
	case structure.AtLeast:
		return "at least " + number(inner.Value)
	case structure.AtMost:
		return "at most " + number(inner.Value)
	case structure.Above:
		return "above " + number(inner.Value)
	case structure.Below:
		return "below " + number(inner.Value)
	case structure.MinLength:
		return "at least " + pluralise(inner.Value, "character")
	case structure.MaxLength:
		return "at most " + pluralise(inner.Value, "character")
	case structure.Pattern:
		return "matching " + inner.Expression
	case structure.MinItems:
		return "at least " + pluralise(inner.Value, "item")
	case structure.MaxItems:
		return "at most " + pluralise(inner.Value, "item")
	default:
		return ""
	}
}

// pluralise words a count, because "at least 1 characters" appears in a file
// other people read.
func pluralise(value int, unit string) string {
	if value == 1 {
		return "1 " + unit
	}
	return strconv.Itoa(value) + " " + unit + "s"
}

func number(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}
