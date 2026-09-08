package unit

// What a source is allowed to have chosen.
//
// The universal representation has one case per kind, but a source that filled
// it may not have had the choice: a format that does not distinguish an integer
// from a number, or a driver that hands character data back as bytes. Reading
// such a value is the schema's decision, not the source's, so these are the
// tolerances stated once and tested here.

import (
	"strings"
	"testing"

	"github.com/mbauer83/effect-golang-schema/schema"
	"github.com/mbauer83/effect-golang-schema/schema/dynamic"
)

// oneField is the smallest description that asks for a value of one kind.
func oneField[A any](name string, of schema.Schema[A]) schema.Schema[dynamic.Value] {
	return schema.Struct[dynamic.Value]("Row", schema.DescribedField(name, of))
}

func rowOf(name string, held dynamic.Value) dynamic.Value {
	return dynamic.Object{Fields: []dynamic.Field{{Name: name, Value: held}}}
}

func TestTextReadsABytesValueTheSourceChoseForIt(t *testing.T) {
	// MySQL hands a varchar column back as bytes and Postgres hands the same
	// column back as a string. The table said text either way, so the schema
	// reads text either way.
	read, err := schema.FromDynamic(
		oneField("site", schema.Text()),
		rowOf("site", dynamic.Bytes{Value: []byte("Kiel")}),
	)
	if err != nil {
		t.Fatal(err)
	}
	object, isObject := read.(dynamic.Object)
	if !isObject {
		t.Fatalf("expected an object, got %#v", read)
	}
	if site, _ := object.Member("site"); site != (dynamic.Text{Value: "Kiel"}) {
		t.Fatalf("expected the characters the bytes held, got %#v", site)
	}
}

func TestTextRefusesBytesThatAreNotText(t *testing.T) {
	// The limit of the tolerance: a binary column read into a text field is a
	// failure, because a string of the broken bytes is not what anybody meant.
	_, err := schema.FromDynamic(
		oneField("site", schema.Text()),
		rowOf("site", dynamic.Bytes{Value: []byte{0xff, 0xfe}}),
	)
	if err == nil {
		t.Fatal("expected bytes that are not text to be refused")
	}
	if !strings.Contains(err.Error(), "expected text at site") {
		t.Fatalf("unexpected refusal: %v", err)
	}
}

func TestAWholeNumberReadsAsAnInteger(t *testing.T) {
	// JSON does not distinguish them, so a buffered document may hold either.
	read, err := schema.FromDynamic(
		oneField("pages", schema.Int()),
		rowOf("pages", dynamic.Number{Value: 632}),
	)
	if err != nil {
		t.Fatal(err)
	}
	object, _ := read.(dynamic.Object)
	if pages, _ := object.Member("pages"); pages != (dynamic.Integer{Value: 632}) {
		t.Fatalf("expected the whole number as an integer, got %#v", pages)
	}
}

func TestANumberThatIsNotWholeIsRefusedAsAnInteger(t *testing.T) {
	_, err := schema.FromDynamic(
		oneField("pages", schema.Int()),
		rowOf("pages", dynamic.Number{Value: 1.5}),
	)
	if err == nil {
		t.Fatal("expected a fractional number to be refused as an integer")
	}
}

func TestTextOfSaysWhichValuesAreText(t *testing.T) {
	// The rule lives here, so a caller reading a row itself applies the same
	// one the schema reader does rather than writing its own.
	for _, held := range []dynamic.Value{
		dynamic.OfText("Kiel"), dynamic.OfBytes([]byte("Kiel")),
	} {
		if text, isText := dynamic.TextOf(held); !isText || text != "Kiel" {
			t.Errorf("%#v did not read as text, got %q", held, text)
		}
	}
	for _, held := range []dynamic.Value{
		dynamic.OfBytes([]byte{0xff, 0xfe}), dynamic.OfInteger(1), dynamic.Absent{},
	} {
		if text, isText := dynamic.TextOf(held); isText {
			t.Errorf("%#v read as text %q", held, text)
		}
	}
}
