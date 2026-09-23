package protobuf

// One scalar on the wire.
//
// The Precision is what decides the layout, which makes this the one place a
// Go-side detail reaches the wire: int32 and int64 are the same varint, but
// float and double are four bytes and eight, and a reader told the wrong one
// reads the wrong value. Everywhere else the Kind is enough.

import (
	"fmt"
	"time"

	"github.com/mbauer83/effect-golang-schema/schema/dynamic"
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// writeScalar writes one writeScalar value under its field number.
//
// present says the field has explicit presence, which decides what happens to a
// zero. proto3 omits the zero of an implicit-presence field -- that is what
// makes a default cost no bytes -- and writes it for one with presence, because
// there the difference between zero and absent is the point.
func writeScalar(
	into *writer,
	number int,
	shape structure.Scalar,
	value dynamic.Value,
	present bool,
) error {
	switch shape.Kind {
	case structure.Text:
		text, ok := value.(dynamic.Text)
		if !ok {
			return kindMismatchError("a string", value)
		}
		if text.Value == "" && !present {
			return nil
		}
		into.block(number, []byte(text.Value))
	case structure.Bytes:
		bytes, ok := value.(dynamic.Bytes)
		if !ok {
			return kindMismatchError("bytes", value)
		}
		if len(bytes.Value) == 0 && !present {
			return nil
		}
		into.block(number, bytes.Value)
	case structure.Boolean:
		flag, ok := value.(dynamic.Boolean)
		if !ok {
			return kindMismatchError("a bool", value)
		}
		if !flag.Value && !present {
			return nil
		}
		into.tag(number, wireVarint)
		into.varint(boolean(flag.Value))
	case structure.Integer:
		return writeInteger(into, number, shape, value, present)
	case structure.Number:
		return writeFractional(into, number, shape, value, present)
	case structure.Timestamp:
		timestamp, ok := value.(dynamic.Timestamp)
		if !ok {
			return kindMismatchError("an instant", value)
		}
		if timestamp.Value.IsZero() && !present {
			return nil
		}
		into.block(number, instant(timestamp.Value))
	default:
		return fmt.Errorf("kind %v has no proto3 form", shape.Kind)
	}
	return nil
}

func writeInteger(
	into *writer,
	number int,
	shape structure.Scalar,
	value dynamic.Value,
	present bool,
) error {
	integer, ok := value.(dynamic.Integer)
	if !ok {
		return kindMismatchError("a whole number", value)
	}
	if integer.Value == 0 && !present {
		return nil
	}
	into.tag(number, wireVarint)
	into.varint(varintOf(integer.Value))
	return nil
}

// varintOf is the varint a whole number becomes.
//
// Two's complement, which sign-extends a negative to sixty-four bits: that is
// what protobuf does, and the reason a negative int32 costs ten bytes on the
// wire. The format's int32 and int64 are the same varint, so the width is a
// statement about range rather than about layout -- which is why the precision
// does not appear here even though it decides the projected type.
func varintOf(value int64) uint64 {
	return uint64(value)
}

func writeFractional(
	into *writer,
	number int,
	shape structure.Scalar,
	value dynamic.Value,
	present bool,
) error {
	fraction, ok := value.(dynamic.Number)
	if !ok {
		return kindMismatchError("a number", value)
	}
	if fraction.Value == 0 && !present {
		return nil
	}
	if shape.Precision == structure.Float32Bits {
		into.tag(number, wireI32)
		into.float(float32(fraction.Value))
		return nil
	}
	into.tag(number, wireI64)
	into.double(fraction.Value)
	return nil
}

// writePackedElement writes one element of a packed field: its value, with no tag.
func writePackedElement(into *writer, shape structure.Scalar, value dynamic.Value) error {
	switch shape.Kind {
	case structure.Boolean:
		flag, ok := value.(dynamic.Boolean)
		if !ok {
			return kindMismatchError("a bool", value)
		}
		into.varint(boolean(flag.Value))
	case structure.Integer:
		integer, ok := value.(dynamic.Integer)
		if !ok {
			return kindMismatchError("a whole number", value)
		}
		into.varint(varintOf(integer.Value))
	case structure.Number:
		number, ok := value.(dynamic.Number)
		if !ok {
			return kindMismatchError("a number", value)
		}
		if shape.Precision == structure.Float32Bits {
			into.float(float32(number.Value))
			return nil
		}
		into.double(number.Value)
	default:
		return fmt.Errorf("kind %v is not packable", shape.Kind)
	}
	return nil
}

// instant is the google.protobuf.Timestamp a time is: seconds in field 1 and
// nanoseconds in field 2, which is the well-known type's own definition.
func instant(moment time.Time) []byte {
	into := &writer{}
	if seconds := moment.Unix(); seconds != 0 {
		into.tag(1, wireVarint)
		into.varint(uint64(seconds))
	}
	if nanos := moment.Nanosecond(); nanos != 0 {
		into.tag(2, wireVarint)
		into.varint(uint64(nanos))
	}
	return into.bytes
}

func boolean(value bool) uint64 {
	if value {
		return 1
	}
	return 0
}

func kindMismatchError(expectation string, value dynamic.Value) error {
	return fmt.Errorf("the description says %s and the value is a %T", expectation, value)
}
