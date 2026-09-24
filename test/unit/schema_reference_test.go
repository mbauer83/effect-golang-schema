package unit

// References between aggregates. What matters is that a reference is the
// target's identity on the wire, that its description says what it
// identifies -- inside a list as well -- and that a field that is not the
// target's identity is refused.

import (
	"strings"
	"testing"

	"github.com/mbauer83/effect-golang-schema/schema"
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

type catalogueFilm struct{ ID int64 }

var catalogueFilmFields = struct {
	ID    schema.Field[catalogueFilm, int64]
	Other schema.Field[catalogueFilm, int64]
}{
	ID:    schema.FieldAt("id", schema.Int64(), func(film *catalogueFilm) *int64 { return &film.ID }).Identity(),
	Other: schema.FieldAt("other", schema.Int64(), func(film *catalogueFilm) *int64 { return &film.ID }),
}

var catalogueFilmSchema = schema.Struct[catalogueFilm]("film", catalogueFilmFields.ID, catalogueFilmFields.Other)

func TestAReferenceIsTheTargetsIdentityAndSaysWhatItIdentifies(t *testing.T) {
	film := schema.Ref(catalogueFilmSchema, catalogueFilmFields.ID)

	encoded, err := schema.EncodeJSON(film, 603)
	if err != nil || strings.TrimSpace(string(encoded)) != "603" {
		t.Fatalf("expected the identity alone, got %s, %v", encoded, err)
	}
	refers := film.Structure().(structure.Scalar).Refers
	if refers == nil || refers.Object != "film" || refers.Key != "id" || refers.OnDelete != schema.Restrict {
		t.Fatalf("expected a restricting reference to film's id, got %+v", refers)
	}
}

func TestAListOfReferencesSaysWhatEachIdentifies(t *testing.T) {
	films := schema.List(schema.Ref(catalogueFilmSchema, catalogueFilmFields.ID, schema.Cascade))
	element := films.Structure().(structure.Sequence).Element.(structure.Scalar)
	if element.Refers == nil || element.Refers.OnDelete != schema.Cascade {
		t.Fatalf("expected each element to refer to a film, cascading, got %+v", element.Refers)
	}
}

func TestAReferenceToAFieldThatIsNotTheIdentityIsRefused(t *testing.T) {
	if err := schema.Validate(schema.Ref(catalogueFilmSchema, catalogueFilmFields.Other)); err == nil ||
		!strings.Contains(err.Error(), "not the identity") {
		t.Fatalf("expected a non-identity refused, got %v", err)
	}
}
