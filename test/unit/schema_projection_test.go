package unit

// A projection publishes a domain type under a surface's own terms. What
// matters is that it leaves out, renames and re-represents exactly the fields
// it names, that what it writes can be read back when nothing is left out, and
// that what cannot be read back says so rather than inventing the missing
// members.

import (
	"strings"
	"testing"

	"github.com/mbauer83/effect-golang-schema/schema"
	"github.com/mbauer83/effect-golang-schema/schema/naming"
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// poster is a value object whose path the domain keeps and a surface
// publishes as a URL.
type poster struct{ Path string }

var posterFields = struct {
	Path schema.Field[poster, string]
}{
	Path: schema.FieldOf("path", schema.Text(),
		func(value poster) string { return value.Path },
		func(value *poster, path string) { value.Path = path }),
}

var posterSchema = schema.Struct[poster]("poster", posterFields.Path)

const imageHost = "https://images.example/w500"

func asURL(path string) string { return imageHost + path }
func asPath(url string) string { return strings.TrimPrefix(url, imageHost) }

func TestAnOmittedFieldIsNeitherWrittenNorDescribed(t *testing.T) {
	titleOnly := filmSchema.Omit(filmFields.Year)
	value, _ := newFilm("Alien", 1979, true)

	encoded, err := schema.EncodeJSON(titleOnly, value)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(encoded)) != `{"title":"Alien"}` {
		t.Fatalf("expected the year left out, got %s", encoded)
	}
	if fields := titleOnly.Structure().(structure.Object).Fields; len(fields) != 1 {
		t.Fatalf("expected the description to lose the year too, got %+v", fields)
	}
}

func TestAnObjectsConstructorIsGivenAnOmittedMemberAsAbsent(t *testing.T) {
	decoded, err := schema.DecodeJSON(filmSchema.Omit(filmFields.Year), []byte(`{"title":"Alien"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, known := decoded.Year(); known {
		t.Fatalf("expected the omitted year to arrive as absent, got %+v", decoded)
	}
	// The constructor still decides: a projection cannot read back what it
	// would refuse.
	if _, err := schema.DecodeJSON(filmSchema.Omit(filmFields.Year), []byte(`{"title":" "}`)); err == nil {
		t.Fatal("expected the constructor's refusal through the projection")
	}
}

func TestAStructProjectionThatLeavesMembersOutRefusesToBeReadBack(t *testing.T) {
	type plain struct{ Title, Note string }
	title := schema.FieldOf("title", schema.Text(), func(value plain) string { return value.Title },
		func(value *plain, text string) { value.Title = text })
	note := schema.FieldOf("note", schema.Text(), func(value plain) string { return value.Note },
		func(value *plain, text string) { value.Note = text })

	_, err := schema.DecodeJSON(schema.Struct[plain]("plain", title, note).Omit(note), []byte(`{"title":"x"}`))
	if err == nil || !strings.Contains(err.Error(), "leaves members out") {
		t.Fatalf("expected the Struct projection to refuse reading back, got %v", err)
	}
}

func TestARenamedFieldKeepsItsNameUnderAStrategy(t *testing.T) {
	renamed := filmSchema.Rename(filmFields.Title, "film_title")
	value, _ := newFilm("Alien", 1979, true)

	encoded, err := schema.EncodeJSON(renamed, value, schema.MemberNaming(naming.CamelCase))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"film_title":"Alien"`) {
		t.Fatalf("expected the given name exactly, got %s", encoded)
	}
	decoded, err := schema.DecodeJSON(renamed, encoded, schema.MemberNaming(naming.CamelCase))
	if err != nil || decoded.Title() != "Alien" {
		t.Fatalf("expected the renamed field to read back, got %+v, %v", decoded, err)
	}
}

func TestARepresentedFieldIsWrittenInItsNewFormAndReadBackInItsOwn(t *testing.T) {
	published := posterSchema.Represent(posterFields.Path, schema.Text(), asURL, asPath)

	encoded, err := schema.EncodeJSON(published, poster{Path: "/matrix.jpg"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"path":"https://images.example/w500/matrix.jpg"`) {
		t.Fatalf("expected the path published as a URL, got %s", encoded)
	}
	decoded, err := schema.DecodeJSON(published, encoded)
	if err != nil || decoded.Path != "/matrix.jpg" {
		t.Fatalf("expected the URL to read back as the path, got %+v, %v", decoded, err)
	}
}

func TestARepresentedFieldStillReachesTheConstructorAsItsOwnType(t *testing.T) {
	// The year is published as text; the constructor still receives an int.
	asText := filmSchema.Represent(filmFields.Year, schema.Text(),
		func(year int) string { return strings.Repeat("I", year%5) },
		func(text string) int { return len(text) })

	decoded, err := schema.DecodeJSON(asText, []byte(`{"title":"Alien","year":"III"}`))
	if err != nil {
		t.Fatal(err)
	}
	if year, known := decoded.Year(); !known || year != 3 {
		t.Fatalf("expected the year converted back before construction, got %d, %v", year, known)
	}
}

func TestANestedObjectIsPublishedThroughItsOwnProjection(t *testing.T) {
	type still struct{ Frame poster }
	frame := schema.FieldOf("frame", posterSchema,
		func(value still) poster { return value.Frame },
		func(value *still, frame poster) { value.Frame = frame })
	stillSchema := schema.Struct[still]("still", frame)

	published := stillSchema.Reshape(frame,
		posterSchema.Represent(posterFields.Path, schema.Text(), asURL, asPath))
	encoded, err := schema.EncodeJSON(published, still{Frame: poster{Path: "/a.jpg"}})
	if err != nil || !strings.Contains(string(encoded), imageHost+"/a.jpg") {
		t.Fatalf("expected the nested path as a URL, got %s, %v", encoded, err)
	}
}

func TestAProjectionSurvivesADescription(t *testing.T) {
	described := filmSchema.WithDescription("a film").Omit(filmFields.Year)
	if err := schema.Validate(described); err != nil {
		t.Fatalf("expected a described object to stay projectable, got %v", err)
	}
	if described.Structure().(structure.Object).Description != "a film" {
		t.Fatal("expected the projection to keep the description")
	}
}

func TestOnlyAnObjectsOwnFieldsCanBeProjected(t *testing.T) {
	if err := schema.Validate(schema.Text().Omit()); err == nil {
		t.Fatal("expected a scalar to refuse a projection")
	}
	// A field of another type does not compile. A field of this type that the
	// object does not declare is refused when the projection is built.
	stranger := schema.FieldOf("title", schema.Text(), film.Title)
	if err := schema.Validate(filmSchema.Rename(stranger, "x")); err == nil {
		t.Fatal("expected a field the object does not declare to be refused")
	}
}

func TestAProjectionDescribesAFieldInItsOwnWords(t *testing.T) {
	published := posterSchema.Represent(posterFields.Path, schema.Text(), asURL, asPath).
		Describe(posterFields.Path, "a whole URL")
	if doc := published.Structure().(structure.Object).Fields[0].Description; doc != "a whole URL" {
		t.Fatalf("expected the projection's description, got %q", doc)
	}
}
