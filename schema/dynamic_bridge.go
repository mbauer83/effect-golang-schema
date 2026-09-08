package schema

// Moving between a Go value and the universal representation.
//
// It needs no new mechanism: the dynamic value's nine cases are the Sink's
// twelve calls seen from the other side, so a sink that builds one and a source
// that reads one are all that is required, and every schema already drives
// both.

import (
	"time"

	"github.com/mbauer83/effect-golang-schema/schema/dynamic"
)

// ToDynamic reads a typed value out as the universal representation, which is
// how a typed schema hands a value to something that only knows the shape.
func ToDynamic[A any](shape Schema[A], value A) (dynamic.Value, error) {
	built := &dynamicSink{}
	if err := Encode(shape, value, built); err != nil {
		return nil, err
	}
	return built.root, nil
}

// FromDynamic builds a typed value from the universal representation, which is
// how a value that arrived as a shape becomes something a Go program can hold.
func FromDynamic[A any](shape Schema[A], value dynamic.Value) (A, error) {
	if value == nil {
		var missing A
		return missing, fail("is missing", nil)
	}
	return Decode(shape, &dynamicSource{pending: []dynamic.Value{value}})
}

// dynamicSink builds a value as a schema writes it.
type dynamicSink struct {
	root  dynamic.Value
	stack []dynamicFrame
}

// dynamicFrame is a container being filled: an object, or a list. field is the
// member name a value is about to be written under.
type dynamicFrame struct {
	object *dynamic.Object
	list   *dynamic.List
	field  string
}

func (sink *dynamicSink) put(value dynamic.Value) error {
	if len(sink.stack) == 0 {
		sink.root = value
		return nil
	}
	frame := &sink.stack[len(sink.stack)-1]
	if frame.object != nil {
		frame.object.Fields = append(frame.object.Fields,
			dynamic.Field{Name: frame.field, Value: value})
		return nil
	}
	frame.list.Elements = append(frame.list.Elements, value)
	return nil
}

func (sink *dynamicSink) Text(value string) error    { return sink.put(dynamic.Text{Value: value}) }
func (sink *dynamicSink) Integer(value int64) error  { return sink.put(dynamic.Integer{Value: value}) }
func (sink *dynamicSink) Number(value float64) error { return sink.put(dynamic.Number{Value: value}) }
func (sink *dynamicSink) Boolean(value bool) error   { return sink.put(dynamic.Boolean{Value: value}) }
func (sink *dynamicSink) Bytes(value []byte) error   { return sink.put(dynamic.Bytes{Value: value}) }
func (sink *dynamicSink) Null() error                { return sink.put(dynamic.Absent{}) }

func (sink *dynamicSink) Timestamp(value time.Time) error {
	return sink.put(dynamic.Timestamp{Value: value})
}

func (sink *dynamicSink) BeginObject() error {
	sink.stack = append(sink.stack, dynamicFrame{object: &dynamic.Object{}})
	return nil
}

func (sink *dynamicSink) FieldName(name string) error {
	sink.stack[len(sink.stack)-1].field = name
	return nil
}

func (sink *dynamicSink) EndObject() error {
	object := sink.stack[len(sink.stack)-1].object
	sink.stack = sink.stack[:len(sink.stack)-1]
	return sink.put(*object)
}

func (sink *dynamicSink) BeginList() error {
	sink.stack = append(sink.stack, dynamicFrame{list: &dynamic.List{Elements: []dynamic.Value{}}})
	return nil
}

func (sink *dynamicSink) EndList() error {
	list := sink.stack[len(sink.stack)-1].list
	sink.stack = sink.stack[:len(sink.stack)-1]
	return sink.put(*list)
}
