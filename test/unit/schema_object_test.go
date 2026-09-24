package unit

// An object built through its constructor. What matters is that a value that
// decodes is one the constructor made -- its rules applied, its refusals
// reported -- and that a field read without being declared is caught when the
// schema is built rather than on every decode.

import (
	"errors"
	"strings"
	"testing"

	"github.com/mbauer83/effect-golang-schema/schema"
)

// film has rules: a title that is not blank, kept trimmed, and a year that may
// be unknown. Its fields are unexported, so nothing but newFilm makes one.
type film struct {
	title string
	year  int
	known bool
}

var errUntitled = errors.New("a film has a title")

func newFilm(title string, year int, known bool) (film, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return film{}, errUntitled
	}
	return film{title: title, year: year, known: known}, nil
}

func (value film) Title() string          { return value.title }
func (value film) Year() (int, bool)      { return value.year, value.known }
func (value film) isSame(other film) bool { return value == other }

var filmFields = struct {
	Title schema.Field[film, string]
	Year  schema.Field[film, int]
}{
	Title: schema.FieldOf("title", schema.Text(), film.Title),
	Year:  schema.OptionalFieldOf("year", schema.Int(), film.Year),
}

var filmSchema = schema.Object("film", func(values schema.Values) (film, error) {
	year, known := filmFields.Year.Lookup(values)
	return newFilm(filmFields.Title.Of(values), year, known)
}, filmFields.Title, filmFields.Year)

func TestADecodedObjectIsOneItsConstructorMade(t *testing.T) {
	decoded, err := schema.DecodeJSON(filmSchema, []byte(`{"title":"  The Matrix ","year":1999}`))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := newFilm("The Matrix", 1999, true)
	if !decoded.isSame(want) {
		t.Fatalf("expected the constructor's trimmed film, got %+v", decoded)
	}
}

func TestAnObjectIsWrittenThroughItsGetters(t *testing.T) {
	value, _ := newFilm("Alien", 0, false)
	encoded, err := schema.EncodeJSON(filmSchema, value)
	if err != nil {
		t.Fatal(err)
	}
	// The absent year is left out, as an absent optional field always is.
	if strings.TrimSpace(string(encoded)) != `{"title":"Alien"}` {
		t.Fatalf("expected the title alone, got %s", encoded)
	}
}

func TestAnAbsentOptionalFieldReadsAsAbsentNotAsZero(t *testing.T) {
	decoded, err := schema.DecodeJSON(filmSchema, []byte(`{"title":"Alien","year":null}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, known := decoded.Year(); known {
		t.Fatalf("expected an unknown year, got %+v", decoded)
	}
}

func TestTheConstructorsRefusalIsTheDecodersRefusal(t *testing.T) {
	_, err := schema.DecodeJSON(filmSchema, []byte(`{"title":"   "}`))
	if err == nil || !strings.Contains(err.Error(), errUntitled.Error()) {
		t.Fatalf("expected the constructor's refusal, got %v", err)
	}
}

func TestAConstructorReadingAnUndeclaredFieldIsRefusedWhenBuilt(t *testing.T) {
	lonely := schema.Object("film", func(values schema.Values) (film, error) {
		// Year is read but not declared: it would be zero on every decode.
		year, known := filmFields.Year.Lookup(values)
		return newFilm(filmFields.Title.Of(values), year, known)
	}, filmFields.Title)

	err := schema.Validate(lonely)
	if err == nil || !strings.Contains(err.Error(), "year") {
		t.Fatalf("expected the undeclared year to be named, got %v", err)
	}
}

func TestAStructFieldWithoutASetterIsRefused(t *testing.T) {
	type plain struct{ Title string }
	settable := schema.Struct[plain]("plain",
		schema.FieldOf("title", schema.Text(), func(value plain) string { return value.Title }))

	if err := schema.Validate(settable); err == nil || !strings.Contains(err.Error(), "setter") {
		t.Fatalf("expected a field without a setter to be refused, got %v", err)
	}
}

func TestAFieldHasAtMostOneSetter(t *testing.T) {
	type plain struct{ Title string }
	set := func(value *plain, title string) { value.Title = title }
	twice := schema.Struct[plain]("plain",
		schema.FieldOf("title", schema.Text(), func(value plain) string { return value.Title }, set, set))

	if err := schema.Validate(twice); err == nil {
		t.Fatal("expected two setters to be refused")
	}
}

func TestAPlainStructFieldIsDeclaredByWhereItIs(t *testing.T) {
	type book struct {
		Title string
		Pages int64
	}
	pages := schema.FieldAt("pages", schema.Int64(), func(value *book) *int64 { return &value.Pages })
	bookSchema := schema.Struct[book]("book",
		schema.FieldAt("title", schema.Text(), func(value *book) *string { return &value.Title }),
		pages)

	encoded, err := schema.EncodeJSON(bookSchema, book{Title: "Dune", Pages: 412})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := schema.DecodeJSON(bookSchema, encoded)
	if err != nil || decoded != (book{Title: "Dune", Pages: 412}) {
		t.Fatalf("expected the book back from %s, got %+v, %v", encoded, decoded, err)
	}
	// A field declared by where it is can be represented another way, as any
	// field can.
	asText := bookSchema.Represent(pages, schema.Text(),
		func(count int64) string { return strings.Repeat("p", int(count%3)) },
		func(text string) int64 { return int64(len(text)) })
	decoded, err = schema.DecodeJSON(asText, []byte(`{"title":"Dune","pages":"pp"}`))
	if err != nil || decoded.Pages != 2 {
		t.Fatalf("expected the represented field set through its address, got %+v, %v", decoded, err)
	}
}
