# effect-golang-schema

A description of a value's shape, and everything that can be read off one.

```go
var BookSchema = schema.Struct[Book]("Book",
    schema.FieldOf("title", schema.MinLength(schema.Text(), 1),
        func(book Book) string { return book.Title },
        func(book *Book, title string) { book.Title = title }),
    schema.FieldOf("pages", schema.AtLeast(schema.Int32(), 1),
        func(book Book) int32 { return book.Pages },
        func(book *Book, pages int32) { book.Pages = pages }),
).Documented("one book on the shelf")
```

One description encodes, decodes, validates, and projects. It is the bottom of
a stack of three modules: [effect-golang-sql](https://github.com/mbauer83/effect-golang-sql)
reads it as tables and migrations, and
[effect-golang-web](https://github.com/mbauer83/effect-golang-web) reads it as
requests, responses, messages and procedures.

**This module carries no third-party dependency, and an architecture test says
so.** A description is data and the formats it projects to are published
specifications, so a program that only describes its shapes acquires nothing by
depending on this. The three dependencies in `go.mod` are test-only reference
implementations, each there to check what a projection emits against something
that has never seen this module.

| Area | State |
|---|---|
| [Schema](docs/reference/schema.md) | usable: shapes, sums (both taggings), constraints, formats, JSON, descriptions with no Go type |
| JSON Schema projection | usable (see the schema reference) |
| [protobuf: proto3 projection and wire codec](docs/reference/protobuf.md) | usable |
| [Derived shapes: create, update, select](docs/reference/variant.md) | usable |
| Generation from a description | usable |

## Layout

```text
schema/                     Schema[A]: shapes, codecs, refinement
  schema/structure/         the description a projection walks
  schema/dynamic/           the value a description carries when there is no Go type
  schema/jsonschema/        the JSON Schema 2020-12 projection
  schema/protobuf/          the proto3 projection and the protobuf wire codec
  schema/variant/           the create, update and select shapes one description has
schemagen/                  writes the Go types a description implies
examples/catalog/           one schema three ways, sequenced in direct style
examples/inventory/         Go types generated from a description
examples/cmd/schemademo/    the examples as a runnable command
test/unit/                  behaviour of the public API
test/acceptance/            the example programs, end to end
test/architecture/          the claims about this module's shape
docs/reference/             what each part is and why it is that way
```

Dependencies point one way: `structure` and `dynamic` know nothing, `schema`
knows those two, and every projection knows `structure`. An architecture test
checks it.

## Running the examples

```sh
go run ./examples/cmd/schemademo
```

## Development

`effect-golang` is not published yet, so `go.mod` resolves it from a sibling
working copy:

```text
workspace/
  effect-golang/            the runtime
  effect-golang-schema/     this module
  effect-golang-sql/        tables and migrations, on this
  effect-golang-web/        transports, on both
```

A `replace` is ignored by anything that depends on *this* module, so it is a
development arrangement and not a distribution one. Replace it with a version
requirement once the runtime is tagged.
