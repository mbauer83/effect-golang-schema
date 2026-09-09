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
	"time"

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

func TestATimestampReadsTextWhenTheSourceHasNoInstantOfItsOwn(t *testing.T) {
	// SQLite has no date type at all, so an instant projected onto it is a
	// text column: a value written as an instant came back as a string that
	// nothing would read, and a whole dialect could not round-trip a
	// timestamp. Which representation a source chose is its business rather
	// than the value's meaning, which is the same argument that lets text
	// read a bytes value.
	wanted := time.Date(2026, 9, 9, 21, 30, 15, 0, time.UTC)
	described := oneField("at", schema.Time())

	for name, carried := range map[string]dynamic.Value{
		"an instant":               dynamic.OfTimestamp(wanted),
		"RFC 3339":                 dynamic.OfText("2026-09-09T21:30:15Z"),
		"RFC 3339 with a fraction": dynamic.OfText("2026-09-09T21:30:15.000Z"),
		"the form SQL prints":      dynamic.OfText("2026-09-09 21:30:15+00:00"),
		"bytes holding one":        dynamic.OfBytes([]byte("2026-09-09T21:30:15Z")),
	} {
		read, err := schema.FromDynamic(described, rowOf("at", carried))
		if err != nil {
			t.Fatalf("expected %s to read as an instant, got %v", name, err)
		}
		held, isInstant := dynamic.TimestampOf(readField(t, read, "at"))
		if !isInstant || !held.Equal(wanted) {
			t.Fatalf("expected %s to be %v, got %v", name, wanted, held)
		}
	}
}

func TestWhatIsNotAnInstantIsRefusedRatherThanGuessedAt(t *testing.T) {
	described := oneField("at", schema.Time())

	for name, carried := range map[string]dynamic.Value{
		"a number":         dynamic.OfInteger(1757451015),
		"a word":           dynamic.OfText("yesterday"),
		"another calendar": dynamic.OfText("09/09/2026"),
		"nothing":          dynamic.OfText(""),
	} {
		if _, err := schema.FromDynamic(described, rowOf("at", carried)); err == nil {
			t.Fatalf("expected %s to be refused", name)
		}
	}
}

// readField is one member of a decoded row, which these tests compare against
// the instant they wrote.
func readField(t *testing.T, row dynamic.Value, name string) dynamic.Value {
	t.Helper()
	object, isObject := row.(dynamic.Object)
	if !isObject {
		t.Fatalf("expected a row, got %#v", row)
	}
	held, present := object.Member(name)
	if !present {
		t.Fatalf("expected a %q member, got %#v", name, row)
	}
	return held
}
