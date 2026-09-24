package unit

// Names written once and spelled by each format's strategy. What matters is
// that a name splits into the same words however it was written, that a codec
// with a strategy writes and reads exactly that spelling, and that a
// description respelled for a projection says what the codec says.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mbauer83/effect-golang-schema/schema"
	"github.com/mbauer83/effect-golang-schema/schema/naming"
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

func TestANameSplitsIntoTheSameWordsHoweverItWasWritten(t *testing.T) {
	for name, want := range map[string][]string{
		"syncedAt":    {"synced", "at"},
		"SyncedAt":    {"synced", "at"},
		"synced_at":   {"synced", "at"},
		"synced-at":   {"synced", "at"},
		"HTTPServer":  {"http", "server"},
		"imdbID":      {"imdb", "id"},
		"w500":        {"w500"},
		"h264Profile": {"h264", "profile"},
		"tmdb_id":     {"tmdb", "id"},
	} {
		if got := naming.Words(name); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: expected %v, got %v", name, want, got)
		}
	}
}

func TestEachStrategySpellsTheWords(t *testing.T) {
	for _, spelling := range []struct {
		strategy naming.Strategy
		want     string
	}{
		{naming.Literal, "imdbID"},
		{naming.CamelCase, "imdbId"},
		{naming.PascalCase, "ImdbId"},
		{naming.SnakeCase, "imdb_id"},
		{naming.KebabCase, "imdb-id"},
	} {
		if got := spelling.strategy.Spell("imdbID"); got != spelling.want {
			t.Errorf("%s: expected %s, got %s", spelling.strategy.Name(), spelling.want, got)
		}
	}
}

type viewing struct {
	FilmID    int64
	WatchedAt string
}

var viewingFilmID = schema.FieldOf("film_id", schema.Int64(),
	func(value viewing) int64 { return value.FilmID },
	func(value *viewing, id int64) { value.FilmID = id })

var viewingSchema = schema.Struct[viewing]("viewing",
	viewingFilmID,
	schema.FieldOf("watched_at", schema.Text(),
		func(value viewing) string { return value.WatchedAt },
		func(value *viewing, at string) { value.WatchedAt = at }),
)

func TestACodecWithAStrategyWritesAndReadsItsSpelling(t *testing.T) {
	camel := schema.MemberNaming(naming.CamelCase)
	encoded, err := schema.EncodeJSON(viewingSchema, viewing{FilmID: 603, WatchedAt: "2026-03-14"}, camel)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(encoded)); got != `{"filmId":603,"watchedAt":"2026-03-14"}` {
		t.Fatalf("expected camelCase members, got %s", got)
	}
	decoded, err := schema.DecodeJSON(viewingSchema, encoded, camel)
	if err != nil || decoded.FilmID != 603 {
		t.Fatalf("expected the camelCase document to read back, got %+v, %v", decoded, err)
	}
}

func TestACodecWithAStrategyReadsNoOtherSpelling(t *testing.T) {
	_, err := schema.DecodeJSON(viewingSchema, []byte(`{"film_id":603,"watched_at":"x"}`),
		schema.MemberNaming(naming.CamelCase))
	if err == nil {
		t.Fatal("expected the snake_case members to be unknown to a camelCase reader")
	}
}

func TestACodecWithoutAStrategyUsesTheNamesAsDeclared(t *testing.T) {
	encoded, err := schema.EncodeJSON(viewingSchema, viewing{FilmID: 1, WatchedAt: "x"})
	if err != nil || !strings.Contains(string(encoded), `"film_id"`) {
		t.Fatalf("expected the declared names, got %s, %v", encoded, err)
	}
}

func TestTwoFieldsAStrategySpellsAlikeAreRefused(t *testing.T) {
	type twins struct{ A, B string }
	clash := schema.Struct[twins]("twins",
		schema.FieldOf("film_id", schema.Text(), func(value twins) string { return value.A },
			func(value *twins, a string) { value.A = a }),
		schema.FieldOf("filmId", schema.Text(), func(value twins) string { return value.B },
			func(value *twins, b string) { value.B = b }),
	)
	_, err := schema.EncodeJSON(clash, twins{}, schema.MemberNaming(naming.SnakeCase))
	if err == nil || !strings.Contains(err.Error(), "film_id") || !strings.Contains(err.Error(), "filmId") {
		t.Fatalf("expected both fields named, got %v", err)
	}
}

// corner is the one variant of a union whose tag field and members are all
// more than one word, so that a strategy has something to respell.
type corner struct{ CornerRadius int64 }

var cornerSchema = schema.Struct[corner]("corner",
	schema.FieldOf("corner_radius", schema.Int64(),
		func(value corner) int64 { return value.CornerRadius },
		func(value *corner, radius int64) { value.CornerRadius = radius }))

var edgeSchema = schema.TaggedUnion[corner]("edge", "edge_kind",
	schema.VariantOf("rounded_corner", cornerSchema,
		func(value corner) (corner, bool) { return value, true },
		func(value corner) corner { return value }))

func TestATaggedUnionsTagAndItsVariantsMembersFollowTheStrategy(t *testing.T) {
	camel := schema.MemberNaming(naming.CamelCase)
	encoded, err := schema.EncodeJSON(edgeSchema, corner{CornerRadius: 4}, camel)
	if err != nil {
		t.Fatal(err)
	}
	// The tag field and the member are respelled; the variant's name is a
	// value, and stays as declared.
	if got := strings.TrimSpace(string(encoded)); got != `{"edgeKind":"rounded_corner","cornerRadius":4}` {
		t.Fatalf("expected the tag field and member in camelCase, got %s", got)
	}
	decoded, err := schema.DecodeJSON(edgeSchema, encoded, camel)
	if err != nil || decoded.CornerRadius != 4 {
		t.Fatalf("expected %s to read back under the same strategy, got %+v, %v", encoded, decoded, err)
	}
}

func TestARespelledDescriptionSaysWhatTheCodecWrites(t *testing.T) {
	spelled := structure.Spell(viewingSchema.Structure(), naming.CamelCase).(structure.Object)
	names := []string{spelled.Fields[0].Name, spelled.Fields[1].Name}
	if !reflect.DeepEqual(names, []string{"filmId", "watchedAt"}) {
		t.Fatalf("expected the members respelled, got %v", names)
	}
	if spelled.Name != "viewing" {
		t.Fatalf("expected the object's own name left to its projection, got %s", spelled.Name)
	}
}

func TestARespelledDescriptionKeepsANameGivenExactly(t *testing.T) {
	renamed := viewingSchema.Rename(viewingFilmID, "tmdb_id")
	spelled := structure.Spell(renamed.Structure(), naming.CamelCase).(structure.Object)
	if spelled.Fields[0].Name != "tmdb_id" || spelled.Fields[1].Name != "watchedAt" {
		t.Fatalf("expected the exact name kept and the other respelled, got %+v", spelled.Fields)
	}
}

func TestASpelledSchemaSpellsItsNamesInEveryFormat(t *testing.T) {
	stored := viewingSchema.Rename(viewingFilmID, "tmdb_id").WithNaming(naming.KebabCase)

	encoded, err := schema.EncodeJSON(stored, viewing{FilmID: 603, WatchedAt: "today"},
		schema.MemberNaming(naming.CamelCase))
	if err != nil {
		t.Fatal(err)
	}
	// The schema's own strategy wins over the reader's, and the exact name
	// stays exact.
	if got := strings.TrimSpace(string(encoded)); got != `{"tmdb_id":603,"watched-at":"today"}` {
		t.Fatalf("expected the schema's spelling, got %s", got)
	}
	value, err := schema.ToDynamic(stored, viewing{FilmID: 1, WatchedAt: "x"})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := schema.FromDynamic(stored, value)
	if err != nil || decoded.FilmID != 1 {
		t.Fatalf("expected the dynamic value to read back, got %+v, %v", decoded, err)
	}
	if fields := stored.Structure().(structure.Object).Fields; fields[1].Name != "watched-at" {
		t.Fatalf("expected the description respelled, got %+v", fields)
	}
}

func TestASpelledTaggedUnionStillReadsBack(t *testing.T) {
	stored := edgeSchema.WithNaming(naming.CamelCase)
	encoded, err := schema.EncodeJSON(stored, corner{CornerRadius: 2})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := schema.DecodeJSON(stored, encoded)
	if err != nil || decoded.CornerRadius != 2 {
		t.Fatalf("expected %s to read back, got %+v, %v", encoded, decoded, err)
	}
}
