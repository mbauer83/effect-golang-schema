package unit

// Constraints. A kind says a value is a number; a constraint says which
// numbers. What matters is that one declaration both refuses a value and
// appears in the description, and that a refusal says which bound it was.

import (
	"strconv"
	"strings"
	"testing"

	"github.com/mbauer83/effect-golang-schema/schema"
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

func TestABoundAdmitsWhatItSaysAndRefusesTheRest(t *testing.T) {
	pages := schema.Int().Constrained(schema.AtLeast[int](1), schema.AtMost[int](100))

	for _, admitted := range []int{1, 50, 100} {
		if _, err := schema.DecodeJSON(pages, []byte(strconv.Itoa(admitted))); err != nil {
			t.Errorf("expected %d to be admitted, got %v", admitted, err)
		}
	}
	for value, reason := range map[int]string{0: "is less than 1", 101: "is greater than 100"} {
		_, err := schema.DecodeJSON(pages, []byte(strconv.Itoa(value)))
		if err == nil {
			t.Errorf("expected %d to be refused", value)
			continue
		}
		if !strings.Contains(err.Error(), reason) {
			t.Errorf("expected %q for %d, got %v", reason, value, err)
		}
	}
}

func TestAnExclusiveBoundExcludesTheBoundItself(t *testing.T) {
	fraction := schema.Float64().Constrained(schema.Above[float64](0), schema.Below[float64](1))

	if _, err := schema.DecodeJSON(fraction, []byte("0.5")); err != nil {
		t.Fatalf("expected 0.5 to be admitted, got %v", err)
	}
	for _, refused := range []string{"0", "1"} {
		if _, err := schema.DecodeJSON(fraction, []byte(refused)); err == nil {
			t.Errorf("expected %s to be refused by an exclusive bound", refused)
		}
	}
}

func TestALengthIsCountedInCharactersRatherThanBytes(t *testing.T) {
	// A length a client can check is the one it can see, and "é" is one
	// character however many bytes it takes.
	short := schema.Text().Constrained(schema.MaxLength(3))

	if _, err := schema.DecodeJSON(short, []byte(`"ééé"`)); err != nil {
		t.Fatalf("expected three characters to be admitted, got %v", err)
	}
	if _, err := schema.DecodeJSON(short, []byte(`"éééé"`)); err == nil {
		t.Fatal("expected four characters to be refused")
	}
}

func TestAPatternRefusesWhatItDoesNotMatch(t *testing.T) {
	code := schema.Text().Constrained(schema.Matching(`^[A-Z]{2}-[0-9]{4}$`))

	if _, err := schema.DecodeJSON(code, []byte(`"AB-1234"`)); err != nil {
		t.Fatalf("expected the code to be admitted, got %v", err)
	}
	_, err := schema.DecodeJSON(code, []byte(`"ab-1234"`))
	if err == nil {
		t.Fatal("expected a code that does not match to be refused")
	}
	if !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected the reason to say so, got %v", err)
	}
}

func TestAnItemCountAppliesToTheList(t *testing.T) {
	authors := schema.List(schema.Text()).Constrained(schema.MinItems[string](1), schema.MaxItems[string](2))

	if _, err := schema.DecodeJSON(authors, []byte(`["one"]`)); err != nil {
		t.Fatalf("expected one author to be admitted, got %v", err)
	}
	for _, refused := range []string{`[]`, `["one","two","three"]`} {
		if _, err := schema.DecodeJSON(authors, []byte(refused)); err == nil {
			t.Errorf("expected %s to be refused", refused)
		}
	}
}

func TestAConstraintIsCheckedOnEncodingToo(t *testing.T) {
	// A value this program built that breaks its own constraint is a mistake
	// here, and finding it at the boundary beats sending it.
	pages := schema.Int().Constrained(schema.AtLeast[int](1))

	if _, err := schema.EncodeJSON(pages, 0); err == nil {
		t.Fatal("expected an out-of-bounds value to be refused before it was written")
	}
}

func TestAConstraintCarriesItsPathThroughAStruct(t *testing.T) {
	type entry struct{ Pages int }
	bounded := schema.Struct[entry]("Entry",
		schema.FieldOf("pages", schema.Int().Constrained(schema.AtLeast[int](1)),
			func(value entry) int { return value.Pages },
			func(value *entry, pages int) { value.Pages = pages }),
	)

	_, err := schema.DecodeJSON(bounded, []byte(`{"pages":0}`))
	if err == nil {
		t.Fatal("expected the bound to refuse the value")
	}
	if path, _ := schema.PathOf(err); strings.Join(path, ".") != "pages" {
		t.Fatalf("expected the path to name the field, got %v from %v", path, err)
	}
}

func TestAConstraintReachesTheDescription(t *testing.T) {
	// One declaration, two jobs: it refuses a value and it appears in the
	// published contract. A constraint that only did the first would leave a
	// client to discover the rule by being rejected.
	shape, isScalar := schema.Int().Constrained(schema.AtLeast[int](1), schema.AtMost[int](100)).
		Structure().(structure.Scalar)
	if !isScalar {
		t.Fatal("expected a scalar")
	}
	if len(shape.Constraints) != 2 {
		t.Fatalf("expected both constraints recorded, got %#v", shape.Constraints)
	}
	// In declared order, so the description reads the way it was written.
	if _, first := shape.Constraints[0].(structure.AtLeast); !first {
		t.Fatalf("expected the lower bound first, got %#v", shape.Constraints)
	}
	if _, second := shape.Constraints[1].(structure.AtMost); !second {
		t.Fatalf("expected the upper bound second, got %#v", shape.Constraints)
	}
}

// objectAsText hands a struct schema to a string combinator, which the compiler
// permits only through a transform. It exists to reach the one case a caller can
// reach: a constraint on a shape that has nowhere to put it.
func objectAsText() schema.Schema[string] {
	return schema.Transform(bookSchema,
		func(book Book) string { return book.Title },
		func(string) Book { return Book{} },
	)
}

func TestConstraintDeclarationMistakesAreReported(t *testing.T) {
	cases := map[string]error{
		// A constraint on an object would have to say which member it meant, so
		// it is refused rather than attached to the first field or dropped.
		"a bound on a struct": schema.Validate(
			objectAsText().Constrained(schema.MinLength(1))),
		"a pattern that does not compile": schema.Validate(
			schema.Text().Constrained(schema.Matching(`[`))),
		"a bound on an unusable schema": schema.Validate(
			schema.Schema[int]{}.Constrained(schema.AtLeast(1))),
	}
	for mistake, err := range cases {
		if err == nil {
			t.Errorf("expected %s to be reported", mistake)
		}
	}
}

func TestAConstraintIsAppliedInTheOrderItIsWritten(t *testing.T) {
	// Two bounds on one string, said left to right. The order matters for the
	// message: a value that breaks both is reported against the first, which
	// is the one a reader of the declaration meets first.
	described := schema.Text().Constrained(schema.MinLength(2), schema.MaxLength(4))
	if err := schema.Validate(described); err != nil {
		t.Fatal(err)
	}
	if _, err := schema.DecodeJSON(described, []byte(`"x"`)); err == nil {
		t.Fatal("expected a string shorter than the minimum to be refused")
	} else if !strings.Contains(err.Error(), "shorter") {
		t.Fatalf("expected the first bound to report, got %v", err)
	}
	if _, err := schema.DecodeJSON(described, []byte(`"xxxxx"`)); err == nil {
		t.Fatal("expected a string longer than the maximum to be refused")
	}
	if held, err := schema.DecodeJSON(described, []byte(`"xxx"`)); err != nil || held != "xxx" {
		t.Fatalf("expected the value between them, got %q %v", held, err)
	}
}

func TestBothBoundsReachTheDescription(t *testing.T) {
	// The other half of what a constraint is for: it appears in the published
	// contract, and applying two of them in one call records both.
	described := schema.Text().Constrained(schema.MinLength(2), schema.MaxLength(4))
	scalar, isScalar := described.Structure().(structure.Scalar)
	if !isScalar {
		t.Fatalf("expected a scalar, got %T", described.Structure())
	}
	if len(scalar.Constraints) != 2 {
		t.Fatalf("expected both constraints recorded, got %+v", scalar.Constraints)
	}
	if _, first := scalar.Constraints[0].(structure.MinLength); !first {
		t.Fatalf("expected the order they were written, got %+v", scalar.Constraints)
	}
}

func TestAConstraintOnAShapeThatTakesNoneIsRefused(t *testing.T) {
	// Every constraint this package builds is about text, a number or a list,
	// so putting one on an object does not compile: there is no
	// Constraint[Holder] to write. What is left is the zero value, which a
	// caller can write and nothing built -- and it is refused rather than
	// quietly ignored, because a declaration that says nothing is a mistake
	// and not a shape.
	described := schema.Struct[holder]("Holder",
		schema.FieldOf("name", schema.Text(),
			func(held holder) string { return held.Name },
			func(held *holder, name string) { held.Name = name }),
	).Constrained(schema.Constraint[holder]{})
	if schema.Validate(described) == nil {
		t.Fatal("expected a constraint an object cannot carry to be refused")
	}
}

type holder struct {
	Name string
}
