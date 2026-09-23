package catalog

import (
	"context"
	"io/fs"

	"github.com/mbauer83/effect-golang-schema/schema"
	"github.com/mbauer83/effect-golang-schema/schema/jsonschema"
	"github.com/mbauer83/effect-golang/effect"
)

type catalogEffect[A any] = effect.Effect[effect.Unit, Fault, A]

// publication is the pair a projection yields: the document to serve and the
// names of the shapes it declares once and refers to thereafter.
type publication struct {
	document   []byte
	components []string
}

// Program loads a catalogue, writes it back normalised, and publishes the
// contract that describes what it accepts.
//
// The schema is validated first, so a declaration mistake fails the program at
// its start rather than on the first document that happens to reach it.
//
// Written in direct style, because the sequence is seven dependent stages and
// each stage's value is what the next one reads: written as a chain, every
// stage would nest inside the one before it.
func Program(inputPath string, normalisedPath string, contractPath string) catalogEffect[Report] {
	io := effect.IOFor[effect.Unit]()
	return effect.Gen(func(do *effect.Do[effect.Unit, Fault]) Report {
		do.Await(validate())

		document := do.Await(read(io, inputPath))
		catalog := do.Await(decode(document))
		encoded := do.Await(normalise(catalog))
		do.Await(write(io, normalisedPath, encoded))

		contract := do.Await(contract())
		do.Await(write(io, contractPath, contract.document))

		return report(catalog, contract)
	}).WithName("catalog")
}

// validate refuses to start on a schema that could never work.
func validate() catalogEffect[effect.Unit] {
	return effect.Try(
		func(context.Context, effect.Unit) (effect.Unit, error) {
			return effect.Unit{}, schema.Validate(Schema)
		},
		faultFor[error]("validating the catalogue schema"),
	).WithName("validate-schema")
}

func read(io effect.IOOperations[effect.Unit], path string) catalogEffect[[]byte] {
	return io.ReadFile(path).
		MapError(faultFor[effect.IOError]("reading the catalogue")).
		WithName("read-catalogue")
}

func write(io effect.IOOperations[effect.Unit], path string, document []byte) catalogEffect[effect.Unit] {
	return io.WriteFile(path, document, fs.FileMode(0o600)).
		MapError(faultFor[effect.IOError]("writing " + path)).
		WithName("write-document")
}

// decode is where a document becomes a value. It is fallible and belongs in
// the same failure channel as the file it came from, which is why it is a stage
// rather than a call.
func decode(document []byte) catalogEffect[Catalog] {
	return effect.Try(
		func(context.Context, effect.Unit) (Catalog, error) {
			return schema.DecodeJSON(Schema, document)
		},
		faultFor[error]("decoding the catalogue"),
	).WithName("decode-catalogue")
}

// normalise writes the decoded value back out. Encoding is deterministic, so
// this is the document a cache or a diff can rely on.
func normalise(catalog Catalog) catalogEffect[[]byte] {
	return effect.Try(
		func(context.Context, effect.Unit) ([]byte, error) {
			return schema.EncodeJSON(Schema, catalog)
		},
		faultFor[error]("normalising the catalogue"),
	).WithName("normalise-catalogue")
}

// contract projects the one description into the published one. No shape is
// declared twice: this is the same Schema the decoder used.
func contract() catalogEffect[publication] {
	return effect.Try(
		func(context.Context, effect.Unit) (publication, error) {
			projection := jsonschema.Project(Schema.Structure())
			document, err := projection.Render()
			if err != nil {
				return publication{}, err
			}
			return publication{document: document, components: projection.ComponentNames()}, nil
		},
		faultFor[error]("publishing the contract"),
	).WithName("publish-contract")
}

func report(catalog Catalog, contract publication) Report {
	onShelf := 0
	for _, book := range catalog.Books {
		if _, inStock := book.Availability.(InStock); inStock {
			onShelf++
		}
	}
	return Report{
		Books:      len(catalog.Books),
		OnShelf:    onShelf,
		Components: contract.components,
	}
}

// faultFor names the stage a failure happened in, so a caller reads one type
// and still reaches the underlying error with errors.As. It is instantiated
// explicitly because the two failure channels it adapts -- a filesystem error
// and a schema error -- are different types at the call site.
func faultFor[E error](stage string) func(E) Fault {
	return func(err E) Fault { return Fault{Stage: stage, Err: err} }
}
