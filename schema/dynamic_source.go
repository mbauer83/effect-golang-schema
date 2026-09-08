package schema

// Reading the universal representation as a schema pulls it.
//
// The other half of the bridge in dynamic_bridge.go, in a file of its own
// because the two directions share nothing but the representation: a sink is
// told what to build, and a source is asked for what it holds.

import (
	"time"

	"github.com/mbauer83/effect-golang-schema/schema/dynamic"
)

// dynamicSource reads a value as a schema pulls it.
//
// pending is a stack rather than a cursor because the schema drives: it asks
// for one value at a time, and a container hands its members over to be asked
// about in turn.
type dynamicSource struct {
	pending []dynamic.Value
}

func (source *dynamicSource) take() (dynamic.Value, error) {
	if len(source.pending) == 0 {
		return nil, fail("the value ended early", nil)
	}
	value := source.pending[len(source.pending)-1]
	source.pending = source.pending[:len(source.pending)-1]
	return value, nil
}

func (source *dynamicSource) push(value dynamic.Value) {
	source.pending = append(source.pending, value)
}

// Text accepts a byte string as well as text, on dynamic.TextOf's terms: which
// representation a source chose for its characters is the source's business,
// and bytes that are not text are still not text.
func (source *dynamicSource) Text() (string, error) {
	value, err := source.take()
	if err != nil {
		return "", err
	}
	text, isText := dynamic.TextOf(value)
	if !isText {
		return "", fail("expected text", nil)
	}
	return text, nil
}

// Integer accepts a number whose value is whole as well as an integer.
//
// A buffered value came from a format that had to guess which of the two a
// number was, and JSON does not distinguish them; the schema does, and it is
// the schema that is asking.
func (source *dynamicSource) Integer() (int64, error) {
	value, err := source.take()
	if err != nil {
		return 0, err
	}
	switch held := value.(type) {
	case dynamic.Integer:
		return held.Value, nil
	case dynamic.Number:
		if whole := int64(held.Value); float64(whole) == held.Value {
			return whole, nil
		}
		return 0, fail("expected a whole number", nil)
	default:
		return 0, fail("expected a whole number", nil)
	}
}

// Number accepts an integer as well as a number, for the reason Integer
// accepts a whole number.
func (source *dynamicSource) Number() (float64, error) {
	value, err := source.take()
	if err != nil {
		return 0, err
	}
	switch held := value.(type) {
	case dynamic.Number:
		return held.Value, nil
	case dynamic.Integer:
		return float64(held.Value), nil
	default:
		return 0, fail("expected a number", nil)
	}
}

func (source *dynamicSource) Boolean() (bool, error) {
	held, err := taken[dynamic.Boolean](source, "a boolean")
	return held.Value, err
}

func (source *dynamicSource) Bytes() ([]byte, error) {
	held, err := taken[dynamic.Bytes](source, "a byte string")
	return held.Value, err
}

func (source *dynamicSource) Timestamp() (time.Time, error) {
	held, err := taken[dynamic.Timestamp](source, "a timestamp")
	return held.Value, err
}

// Null consumes the value only when it is one, because asking is not the same
// as reading and a present value must still be there afterwards.
func (source *dynamicSource) Null() (bool, error) {
	if len(source.pending) == 0 {
		return false, fail("the value ended early", nil)
	}
	if _, absent := source.pending[len(source.pending)-1].(dynamic.Absent); !absent {
		return false, nil
	}
	_, err := source.take()
	return true, err
}

func (source *dynamicSource) ReadObject(decode func(name string) error) error {
	object, err := taken[dynamic.Object](source, "an object")
	if err != nil {
		return err
	}
	for _, field := range object.Fields {
		source.push(field.Value)
		if err := decode(field.Name); err != nil {
			return err
		}
	}
	return nil
}

func (source *dynamicSource) ReadList(decode func() error) error {
	list, err := taken[dynamic.List](source, "a list")
	if err != nil {
		return err
	}
	for _, element := range list.Elements {
		source.push(element)
		if err := decode(); err != nil {
			return err
		}
	}
	return nil
}

func (source *dynamicSource) Skip() error {
	_, err := source.take()
	return err
}

// Buffer hands over the whole value, which for this source costs nothing: it
// is already a value.
func (source *dynamicSource) Buffer() (dynamic.Value, error) {
	return source.take()
}

// taken reads the next value and checks it is the case the schema asked for.
func taken[A dynamic.Value](source *dynamicSource, wanted string) (A, error) {
	var missing A
	value, err := source.take()
	if err != nil {
		return missing, err
	}
	held, is := value.(A)
	if !is {
		return missing, fail("expected "+wanted, nil)
	}
	return held, nil
}
