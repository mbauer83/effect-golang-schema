package catalog

import (
	"context"
	"io/fs"

	"github.com/mbauer83/effect-golang-schema/schema"
	"github.com/mbauer83/effect-golang-schema/schema/jsonschema"
	"github.com/mbauer83/effect-golang/effect"
)

type catalogEffect[A any] = effect.Effect[effect.Unit, Fault, A]

// published is the pair a projection yields: the document to serve and the
// names of the shapes it declares once and refers to thereafter.
type published struct {
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
		do.Await(validated())

		document := do.Await(read(io, inputPath))
		catalog := do.Await(decoded(document))
		encoded := do.Await(normalised(catalog))
		do.Await(write(io, normalisedPath, encoded))

		contract := do.Await(contract())
		do.Await(write(io, contractPath, contract.document))

		return report(catalog, contract)
	}).WithName("catalog")
}

// validated refuses to start on a schema that could never work.
func validated() catalogEffect[effect.Unit] {
	return effect.Try(
		func(context.Context, effect.Unit) (effect.Unit, error) {
			return effect.Unit{}, schema.Validate(Schema)
		},
		faulting[error]("validating the catalogue schema"),
	).WithName("validate-schema")
}

func read(io effect.IOOperations[effect.Unit], path string) catalogEffect[[]byte] {
	return io.ReadFile(path).
		MapError(faulting[effect.IOError]("reading the catalogue")).
		WithName("read-catalogue")
}

func write(io effect.IOOperations[effect.Unit], path string, document []byte) catalogEffect[effect.Unit] {
	return io.WriteFile(path, document, fs.FileMode(0o600)).
		MapError(faulting[effect.IOError]("writing " + path)).
		WithName("write-document")
}

// decoded is where a document becomes a value. It is fallible and belongs in
// the same failure channel as the file it came from, which is why it is a stage
// rather than a call.
func decoded(document []byte) catalogEffect[Catalog] {
	return effect.Try(
		func(context.Context, effect.Unit) (Catalog, error) {
			return schema.DecodeJSON(Schema, document)
		},
		faulting[error]("decoding the catalogue"),
	).WithName("decode-catalogue")
}

// normalised writes the decoded value back out. Encoding is deterministic, so
// this is the document a cache or a diff can rely on.
func normalised(catalog Catalog) catalogEffect[[]byte] {
	return effect.Try(
		func(context.Context, effect.Unit) ([]byte, error) {
			return schema.EncodeJSON(Schema, catalog)
		},
		faulting[error]("normalising the catalogue"),
	).WithName("normalise-catalogue")
}

// contract projects the one description into the published one. No shape is
// declared twice: this is the same Schema the decoder used.
func contract() catalogEffect[published] {
	return effect.Try(
		func(context.Context, effect.Unit) (published, error) {
			projected := jsonschema.Project(Schema.Structure())
			document, err := projected.Render()
			if err != nil {
				return published{}, err
			}
			return published{document: document, components: projected.ComponentNames()}, nil
		},
		faulting[error]("publishing the contract"),
	).WithName("publish-contract")
}

func report(catalog Catalog, contract published) Report {
	shelved := 0
	for _, book := range catalog.Books {
		if _, onTheShelf := book.Availability.(InStock); onTheShelf {
			shelved++
		}
	}
	return Report{
		Books:      len(catalog.Books),
		Shelved:    shelved,
		Components: contract.components,
	}
}

// faulting names the stage a failure happened in, so a caller reads one type
// and still reaches the underlying error with errors.As. It is instantiated
// explicitly because the two failure channels it adapts -- a filesystem error
// and a schema error -- are different types at the call site.
func faulting[E error](stage string) func(E) Fault {
	return func(err E) Fault { return Fault{Stage: stage, Err: err} }
}
