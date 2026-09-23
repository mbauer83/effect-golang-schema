package schema

// The standard string formats.
//
// A format is two different claims and this package makes both. It is an
// annotation a reader of the contract acts on -- JSON Schema's own format
// keyword asserts nothing, by design -- and it is a rule a server has to
// enforce, because a request is refused here or it is not refused at all.
//
// Where the rule is a regular expression, the expression is recorded as well as
// the format name, so a consumer whose validator ignores format still gets the
// check from pattern. Where the rule is arithmetic or grammar -- an address, a
// URI -- there is no expression to record, and the document can only annotate.
// That asymmetry is real and is not hidden.

import (
	"net/mail"
	"net/netip"
	"net/url"

	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// UUID admits the textual form of a UUID, in any case.
func UUID() Schema[string] {
	return TextFormat("uuid").Check(Pattern(
		`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`))
}

// Email admits one addr-spec, which is what a form field carries.
//
// It is parsed rather than matched, because the grammar is not a regular
// language and every regular expression that claims to be it is wrong about
// something. A display name is refused: "Ada <ada@example.test>" is a mailbox,
// not an address, and a field asking for an address means the address.
func Email() Schema[string] {
	return withCheck(TextFormat("email"), func(value string) error {
		address, err := mail.ParseAddress(value)
		if err != nil {
			return fail("is not an email address", err)
		}
		if address.Address != value {
			return fail("is a mailbox rather than an address", nil)
		}
		return nil
	})
}

// URI admits an absolute URI: one that says what scheme it is.
//
// A relative reference is a legitimate thing and a different one, so it has its
// own constructor rather than being quietly admitted here.
func URI() Schema[string] {
	return withCheck(TextFormat("uri"), func(value string) error {
		uri, err := url.Parse(value)
		if err != nil {
			return fail("is not a URI", err)
		}
		if !uri.IsAbs() {
			return fail("is a relative reference rather than a URI", nil)
		}
		return nil
	})
}

// URL admits an absolute URI that locates something: a scheme, "://", and a
// host.
//
// mailto:ada@example.test is a URI and is not a URL, which is the whole
// difference between naming a thing and saying where it is. The rule is an
// expression, so the document carries it and a consumer's validator enforces
// the same thing this one does.
//
// It annotates as "uri", because that is the registered format name and there
// is no registered one for a locator.
func URL() Schema[string] {
	return TextFormat("uri").Check(Pattern(`^[A-Za-z][A-Za-z0-9+.\-]*://[^/?#]+`))
}

// URIReference admits a URI or a relative reference.
func URIReference() Schema[string] {
	return withCheck(TextFormat("uri-reference"), func(value string) error {
		if _, err := url.Parse(value); err != nil {
			return fail("is not a URI reference", err)
		}
		return nil
	})
}

// Hostname admits a host name by the RFC 1123 rules: labels of letters, digits
// and hyphens, each at most 63 characters, and 253 in total.
func Hostname() Schema[string] {
	// The expression already refuses an empty name and a label that starts or
	// ends with a hyphen; the length is the one rule it cannot state.
	return TextFormat("hostname").Check(
		Pattern(`^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?)*$`),
		MaxLength(253))
}

// IPv4 admits a dotted-quad address.
func IPv4() Schema[string] {
	return withCheck(TextFormat("ipv4"), func(value string) error {
		return checkAddress(value, 4, "is not an IPv4 address")
	})
}

// IPv6 admits an IPv6 address, in any of its written forms.
func IPv6() Schema[string] {
	return withCheck(TextFormat("ipv6"), func(value string) error {
		return checkAddress(value, 16, "is not an IPv6 address")
	})
}

// checkAddress parses an address and checks its family. The family is told from the
// parsed form rather than from the text, because a dotted quad is also a valid
// IPv6 address written the short way, and net.ParseIP admits both.
func checkAddress(value string, width int, reason string) error {
	address, err := netip.ParseAddr(value)
	if err != nil {
		return fail(reason, err)
	}
	if len(address.AsSlice()) != width {
		return fail(reason, nil)
	}
	return nil
}

// withCheck narrows a schema with a rule the description cannot state.
//
// The value is withCheck in both directions, as a constraint is, but nothing is
// added to the structure: there is no keyword for "parses as an address", and
// inventing one would say something no other projection could read.
func withCheck[A any](inner Schema[A], check func(A) error) Schema[A] {
	if fault := Validate(inner); fault != nil {
		return faultySchema[A](inner.node, fault)
	}
	return of(
		inner.node,
		func(value A, into Sink) error {
			if err := check(value); err != nil {
				return err
			}
			return Encode(inner, value, into)
		},
		func(from Source) (A, error) {
			value, err := Decode(inner, from)
			if err != nil {
				return value, err
			}
			if err := check(value); err != nil {
				var zero A
				return zero, err
			}
			return value, nil
		},
	)
}

// Constructor is what a generator needs to write a scalar back out: the call,
// and how many of the constraints in the shape that call already carries.
//
// The count is here rather than in the generator because only this package
// knows what each constructor records, and a second table would fall out of
// step with these the first time one of them changed.
type Constructor struct {
	Call            string
	ConstraintCount int
	// GoType is the Go type the call describes, because a generator emitting a
	// bound has to name it: a constraint is a value now rather than a wrapper,
	// so there is no inner schema for the compiler to read the type from.
	GoType string
}

// FormatConstructors names the constructor for each standard format.
var FormatConstructors = map[string]Constructor{
	"uuid":          {Call: "schema.UUID()", ConstraintCount: countConstraints(UUID()), GoType: "string"},
	"email":         {Call: "schema.Email()", ConstraintCount: countConstraints(Email()), GoType: "string"},
	"uri":           {Call: "schema.URI()", ConstraintCount: countConstraints(URI()), GoType: "string"},
	"uri-reference": {Call: "schema.URIReference()", ConstraintCount: countConstraints(URIReference()), GoType: "string"},
	"hostname":      {Call: "schema.Hostname()", ConstraintCount: countConstraints(Hostname()), GoType: "string"},
	"ipv4":          {Call: "schema.IPv4()", ConstraintCount: countConstraints(IPv4()), GoType: "string"},
	"ipv6":          {Call: "schema.IPv6()", ConstraintCount: countConstraints(IPv6()), GoType: "string"},
}

// PrecisionConstructors names the constructor for each Go numeric width.
var PrecisionConstructors = map[structure.Precision]Constructor{
	structure.Int8Bits:    {Call: "schema.Int8()", ConstraintCount: countConstraints(Int8()), GoType: "int8"},
	structure.Int16Bits:   {Call: "schema.Int16()", ConstraintCount: countConstraints(Int16()), GoType: "int16"},
	structure.Int32Bits:   {Call: "schema.Int32()", ConstraintCount: countConstraints(Int32()), GoType: "int32"},
	structure.Int64Bits:   {Call: "schema.Int64()", ConstraintCount: countConstraints(Int64()), GoType: "int64"},
	structure.IntBits:     {Call: "schema.Int()", ConstraintCount: countConstraints(Int()), GoType: "int"},
	structure.Uint8Bits:   {Call: "schema.Uint8()", ConstraintCount: countConstraints(Uint8()), GoType: "uint8"},
	structure.Uint16Bits:  {Call: "schema.Uint16()", ConstraintCount: countConstraints(Uint16()), GoType: "uint16"},
	structure.Uint32Bits:  {Call: "schema.Uint32()", ConstraintCount: countConstraints(Uint32()), GoType: "uint32"},
	structure.Uint64Bits:  {Call: "schema.Uint64()", ConstraintCount: countConstraints(Uint64()), GoType: "uint64"},
	structure.UintBits:    {Call: "schema.Uint()", ConstraintCount: countConstraints(Uint()), GoType: "uint"},
	structure.Float32Bits: {Call: "schema.Float32()", ConstraintCount: countConstraints(Float32()), GoType: "float32"},
	structure.Float64Bits: {Call: "schema.Float64()", ConstraintCount: countConstraints(Float64()), GoType: "float64"},
}

// countConstraints counts what a constructor records, so a generator emitting the
// constructor knows not to emit those again.
func countConstraints[A any](shape Schema[A]) int {
	scalar, isScalar := shape.node.(structure.Scalar)
	if !isScalar {
		return 0
	}
	return len(scalar.Constraints)
}
