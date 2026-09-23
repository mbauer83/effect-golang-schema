// Package definitions describes the inventory's shapes, and is the source of
// truth for them.
//
// These are schemas, not a separate declaration language: the same Struct, the
// same OneOf, the same constraints. What they leave out is the getters and
// setters, which are the only part of a field declaration that needs a Go type
// -- so a description compiles before the type it will become exists, and the
// Go types are generated from it rather than written a second time.
//
// The descriptions are unexported. They are input to generation, and the
// application uses the generated package: two usable schemas for one shape
// would be one too many, and the typed one is the one a program wants.
//
// The widths are stated here and not left to the generator. A description that
// only said "a whole number" would make the generator guess, and a published
// contract would claim a range wider than the program accepts.
package definitions

import (
	"github.com/mbauer83/effect-golang-schema/schema"
	"github.com/mbauer83/effect-golang-schema/schema/dynamic"
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// item is one stocked line.
var item = schema.Struct[dynamic.Value]("Item",
	schema.DynamicField("sku", schema.Text().Check(schema.Pattern(`^[A-Z]{3}-[0-9]{5}$`))).
		WithDescription("the stock-keeping unit, three letters and five digits"),
	schema.DynamicField("onHand", schema.Uint16()).
		WithDescription("how many are on hand"),
	schema.DynamicField("weightGrams", schema.Float32()).
		WithDescription("what one unit weighs"),
	schema.DynamicField("id", schema.UUID()),
	schema.DynamicField("tags", schema.List(schema.Text()).Check(schema.MinItems[string](1))),
	schema.DynamicField("note", schema.Text().Check(schema.MaxLength(200))).Optional(),
).WithDescription("one stocked line")

// movement is a change in what is stocked.
var movement = schema.OneOf[dynamic.Value]("Movement",
	schema.DynamicVariant("received", schema.Struct[dynamic.Value]("Received",
		schema.DynamicField("count", schema.Uint16()),
	)).WithDescription("stock arriving"),
	schema.DynamicVariant("shipped", schema.Struct[dynamic.Value]("Shipped",
		schema.DynamicField("count", schema.Uint16()),
		schema.DynamicField("to", schema.Hostname()),
	)).WithDescription("stock leaving"),
).WithDescription("a change in what is stocked")

// Descriptions are every shape to generate, in the order to write them.
func Descriptions() []structure.Node {
	return []structure.Node{item.Structure(), movement.Structure()}
}

// Faults reports a description that could not be used, so a mistake in one is a
// start-up error here rather than a puzzling generator failure.
func Faults() []error {
	faults := []error{}
	for _, description := range []schema.Schema[dynamic.Value]{item, movement} {
		if err := schema.Validate(description); err != nil {
			faults = append(faults, err)
		}
	}
	return faults
}
