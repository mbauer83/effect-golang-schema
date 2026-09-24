package catalog

import (
	"time"

	"github.com/mbauer83/effect-golang-schema/schema"
)

// CatalogSchema describes Catalog, binding the Go type to the wire.
var CatalogSchema = schema.Struct[Catalog]("Catalog",
	schema.FieldAt("name", schema.Text(),
		func(value *Catalog) *string { return &value.Name }).WithDescription("Name identifies the catalogue."),
	schema.FieldAt("books", schema.List(BookSchema),
		func(value *Catalog) *[]Book { return &value.Books }).WithDescription("Books are every entry, in the order the catalogue lists them."),
).WithDescription("Catalog is what one document holds.")

// BookSchema describes Book, binding the Go type to the wire.
var BookSchema = schema.Struct[Book]("Book",
	schema.FieldAt("title", schema.Text().Check(schema.MinLength(1)),
		func(value *Book) *string { return &value.Title }).WithDescription("Title is what the book is called."),
	schema.FieldAt("authors", schema.List(schema.Text()),
		func(value *Book) *[]string { return &value.Authors }).WithDescription("Authors are credited in the order the book credits them."),
	schema.FieldAt("pages", schema.Int().Check(schema.AtLeast[int](1), schema.AtMost[int](20000)),
		func(value *Book) *int { return &value.Pages }).WithDescription("Pages is how many pages the book has, and there is at least one."),
	schema.OptionalFieldOf("subtitle", schema.Text(),
		func(value Book) (string, bool) {
			var absent string
			if value.Subtitle == nil {
				return absent, false
			}
			return *value.Subtitle, true
		},
		func(value *Book, field string) { value.Subtitle = &field }).WithDescription("Subtitle is absent for a book that has none."),
	schema.FieldAt("availability", AvailabilitySchema,
		func(value *Book) *Availability { return &value.Availability }).WithDescription("Availability is whether the book can be had."),
).WithDescription("Book is one entry.")

// InStockSchema describes InStock, binding the Go type to the wire.
var InStockSchema = schema.Struct[InStock]("InStock",
	schema.FieldAt("count", schema.Int(),
		func(value *InStock) *int { return &value.Count }).WithDescription("Count is how many copies are on the shelf."),
).WithDescription("InStock is a title on the shelf, with how many copies.")

// OnOrderSchema describes OnOrder, binding the Go type to the wire.
var OnOrderSchema = schema.Struct[OnOrder]("Awaited",
	schema.FieldAt("expected", schema.Time(),
		func(value *OnOrder) *time.Time { return &value.DueDate }).WithDescription("DueDate is when the title should arrive."),
).WithDescription("Awaited is a title not yet arrived, with when it is expected.")

// DiscontinuedSchema describes Discontinued, binding the Go type to the wire.
var DiscontinuedSchema = schema.Struct[Discontinued]("Discontinued").WithDescription("Discontinued is a title that will not be restocked.")
