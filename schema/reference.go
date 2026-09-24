package schema

// A reference to another aggregate, by its identity.

import (
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// What deleting a referenced object does to a reference to it.
const (
	// Restrict refuses to delete an object something still refers to.
	Restrict = structure.Restrict
	// Cascade deletes what refers to it as well.
	Cascade = structure.Cascade
	// SetNull empties the reference; only an optional one can be emptied.
	SetNull = structure.SetNull
)

// Ref is a reference to another aggregate: a value of the target's identity,
// described as the identity of that target.
//
// On the wire it is the identity and nothing more. What it adds is what a
// projection to storage needs to keep it honest -- a foreign key, and what
// deleting the target does, Restrict unless stated -- and it says "a film"
// where the identity's own shape would say "a number".
//
//	FilmRef: schema.FieldOf("film", schema.Ref(catalog.FilmSchema, catalog.FilmFields.ID), Viewing.Film)
//
// The identity must be target's own: a field it declares as its Identity.
func Ref[T, ID any](target Schema[T], identity Field[T, ID], onDelete ...structure.Deletion) Schema[ID] {
	shape := identity.shape
	parts, object, fault := target.projectable()
	if fault != nil {
		return faultySchema[ID](shape.node, fault)
	}
	if index := parts.index(identity.erased.key); index < 0 || !parts.fields[index].identity {
		return faultySchema[ID](shape.node,
			fail(identity.erased.name+" is not the identity of "+object.Name, nil))
	}
	scalar, isScalar := shape.node.(structure.Scalar)
	if !isScalar {
		return faultySchema[ID](shape.node, fail("a reference identifies by one value", nil))
	}
	deletion := Restrict
	switch len(onDelete) {
	case 0:
	case 1:
		deletion = onDelete[0]
	default:
		return faultySchema[ID](shape.node, fail("a reference says once what deletion does", nil))
	}
	scalar.Refers = &structure.Target{Object: object.Name, Key: identity.erased.name, OnDelete: deletion}
	shape.node = scalar
	return shape
}
