package unit

// The TypeScript projection. What matters is that every named shape is
// declared once and before it is used, that a recursive one is suspended, and
// that both wire forms of a union come out as what arrives.

import (
	"strings"
	"testing"

	"github.com/mbauer83/effect-golang-schema/schema"
	"github.com/mbauer83/effect-golang-schema/schema/typescript"
)

func TestANamedShapeBecomesASchemaAndAType(t *testing.T) {
	module, err := typescript.Module("", bookSchema.Structure())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`import { Schema } from "effect";`,
		"export const Book = Schema.Struct({",
		"  title: Schema.String,",
		"  authors: Schema.Array(Schema.String),",
		"  pages: Schema.Number,",
		"  subtitle: Schema.optionalKey(Schema.String),",
		"  hasIndex: Schema.Boolean,",
		"export type Book = typeof Book.Type;",
	} {
		if !strings.Contains(module, want) {
			t.Errorf("expected %q in\n%s", want, module)
		}
	}
}

func TestARecursiveShapeIsSuspendedRatherThanUsedBeforeItIsDeclared(t *testing.T) {
	type Node struct {
		Label    string
		Children []Node
	}
	var nodeSchema schema.Schema[Node]
	nodeSchema = schema.Struct[Node]("Node",
		schema.FieldOf("label", schema.Text(),
			func(node Node) string { return node.Label },
			func(node *Node, label string) { node.Label = label }),
		schema.FieldOf("children", schema.List(schema.Suspend(func() schema.Schema[Node] {
			return nodeSchema
		})),
			func(node Node) []Node { return node.Children },
			func(node *Node, children []Node) { node.Children = children }),
	)

	module, err := typescript.Module("", nodeSchema.Structure())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		// TypeScript cannot infer a type through a value that refers to
		// itself, so the type is written out and the value annotated with it.
		"export type Node = { readonly label: string; readonly children: ReadonlyArray<Node>; };",
		"export const Node: Schema.Codec<Node> = Schema.Struct({",
		"children: Schema.Array(Schema.suspend((): Schema.Codec<Node> => Node)),",
	} {
		if !strings.Contains(module, want) {
			t.Errorf("expected %q in\n%s", want, module)
		}
	}
}

func TestAUnionToldApartByAFieldCarriesItAsALiteral(t *testing.T) {
	module, err := typescript.Module("", toleranceSchema.Structure())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(module, "export const Tolerance = Schema.Union([") ||
		!strings.Contains(module, `type: Schema.Literal("iso2768"),`) {
		t.Fatalf("expected the variants tagged by their field, got\n%s", module)
	}
}

func TestTwoDifferentShapesWithOneNameAreRefused(t *testing.T) {
	type first struct{ A string }
	type second struct{ B string }
	one := schema.Struct[first]("Same",
		schema.FieldOf("a", schema.Text(),
			func(value first) string { return value.A },
			func(value *first, a string) { value.A = a }))
	other := schema.Struct[second]("Same",
		schema.FieldOf("b", schema.Text(),
			func(value second) string { return value.B },
			func(value *second, b string) { value.B = b }))

	if _, err := typescript.Module("", one.Structure(), other.Structure()); err == nil {
		t.Fatal("expected two shapes named Same to be refused")
	}
}

func TestAComponentIsNamedInPascalCaseWhateverCaseTheSchemaUsed(t *testing.T) {
	type row struct{ Title string }
	cardRow := schema.Struct[row]("card_row",
		schema.FieldOf("title_text", schema.Text(),
			func(value row) string { return value.Title },
			func(value *row, title string) { value.Title = title }))

	module, err := typescript.Module("", cardRow.Structure())
	if err != nil {
		t.Fatal(err)
	}
	// The type is TypeScript's to name; the key is the wire's, and stays.
	if !strings.Contains(module, "export const CardRow = Schema.Struct({") ||
		!strings.Contains(module, "  title_text: Schema.String,") {
		t.Fatalf("expected a PascalCase component with its wire key, got\n%s", module)
	}
}
