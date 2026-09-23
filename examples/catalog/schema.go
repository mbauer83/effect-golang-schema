package catalog

import "github.com/mbauer83/effect-golang-schema/schema"

// Schema describes a whole catalogue document.
//
// It is a package-level value because a schema is immutable and reusable:
// building it once and validating it at start-up is the whole point of
// separating description from use.
var Schema = CatalogSchema

// AvailabilitySchema names the alternative it carries, so a decoder reading
// {"inStock": {...}} knows which shape follows before it reads it.
//
// A union is written by hand: which Go types are the alternatives, and how to
// narrow to each, is not in the interface's declaration.
var AvailabilitySchema = schema.OneOf[Availability]("Availability",
	schema.VariantOf("inStock", InStockSchema,
		func(availability Availability) (InStock, bool) {
			stock, is := availability.(InStock)
			return stock, is
		},
		func(stock InStock) Availability { return stock }).
		WithDescription("on the shelf, with how many copies"),
	schema.VariantOf("awaited", OnOrderSchema,
		func(availability Availability) (OnOrder, bool) {
			onOrder, is := availability.(OnOrder)
			return onOrder, is
		},
		func(onOrder OnOrder) Availability { return onOrder }).
		WithDescription("not yet arrived, with when it is expected"),
	schema.VariantOf("discontinued", DiscontinuedSchema,
		func(availability Availability) (Discontinued, bool) {
			gone, is := availability.(Discontinued)
			return gone, is
		},
		func(gone Discontinued) Availability { return gone }).
		WithDescription("will not be restocked"),
)
