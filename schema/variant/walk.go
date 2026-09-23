package variant

// The walk that leaves fields out.
//
// It rebuilds the tree rather than mutating it, because a description is shared:
// two endpoints may hold the same one, and a derivation that edited it in place
// would change what the other publishes.

import (
	"fmt"

	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

func deriveNode(node structure.Node, policy keepPolicy) (structure.Node, error) {
	if reference, isReference := node.(structure.Reference); isReference && reference.Resolve == nil {
		// A name with nothing behind it cannot be derived from, and passing it
		// through would publish a shape whose contents nobody can see.
		return nil, fmt.Errorf("%q: %w", reference.Name, ErrUnresolved)
	}
	object, isObject := resolveObject(node)
	if !isObject {
		return nil, ErrNotAnObject
	}

	fields := make([]structure.Field, 0, len(object.Fields))
	for _, field := range object.Fields {
		survivor, keep, err := deriveField(field, policy)
		if err != nil {
			return nil, fmt.Errorf("field %q of %s: %w", field.Name, object.Name, err)
		}
		if keep {
			fields = append(fields, survivor)
		}
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("%s: %w", object.Name, ErrNothingLeft)
	}

	return structure.Object{Name: object.Name, Description: object.Description, Fields: fields}, nil
}

// deriveField decides one field's fate, and what it looks like if it survives.
func deriveField(
	field structure.Field,
	policy keepPolicy,
) (structure.Field, bool, error) {
	switch {
	case field.Computed:
		return structure.Field{}, false, nil
	case field.Identity && !policy.keepIdentity:
		return structure.Field{}, false, nil
	}

	entity, nested := entityInside(field.Node)
	if nested && !policy.reachIntoEntities {
		return structure.Field{}, false, nil
	}
	if nested {
		// The entity's own fields are derived the same way, so a computed
		// column on a child is left out of the child's shape too.
		inner, err := deriveWithin(field.Node, entity, policy)
		if err != nil {
			return structure.Field{}, false, err
		}
		field.Node = inner
	}

	// The marks are the description's, not the derived shape's: a shape a
	// caller supplies has no identity to declare and nothing computed left in
	// it, so carrying the marks through would say something untrue about it.
	field.Identity = false
	field.Computed = false
	if policy.partial {
		field.Optional = true
	}
	return field, true, nil
}

// deriveWithin derives a nested entity, keeping whatever wraps it.
//
// A list of order lines is a list of a derived order line, and a nullable one is
// a nullable derived one. The wrapper survives because it says how many there
// are and whether there is one, which a derivation has no business changing.
func deriveWithin(
	node structure.Node,
	entity structure.Object,
	policy keepPolicy,
) (structure.Node, error) {
	// An entity nested inside another is created with its parent, so its own
	// identity is its to supply -- but it is never *updated* through the
	// parent, because it has an identity of its own to be selected by. Keeping
	// the outer decision is what makes that fall out.
	inner, err := deriveNode(entity, policy)
	if err != nil {
		return nil, err
	}
	return rewrapNode(node, inner)
}

// rewrapNode puts a derived element back inside whatever held the original.
func rewrapNode(node structure.Node, inner structure.Node) (structure.Node, error) {
	switch shape := node.(type) {
	case structure.Object:
		return inner, nil
	case structure.Reference:
		return inner, nil
	case structure.Sequence:
		return structure.Sequence{Element: inner, Constraints: shape.Constraints}, nil
	case structure.Nullable:
		return structure.Nullable{Inner: inner}, nil
	case structure.Mapping:
		return structure.Mapping{Key: shape.Key, Value: inner}, nil
	default:
		return nil, fmt.Errorf("%T holds an entity and this walk cannot rebuild it", node)
	}
}

// entityInside is the entity a field carries, through whatever wraps it,
// including a map.
//
// Deliberately not structure.EntityBehind, and the difference is the map. That
// one answers the storage question -- does this field get a table -- and a map
// of entities does not, because its key would need somewhere of its own to live
// and the description does not name it. This answers a different question:
// whether the caller creates these separately. A map of entities is still a map
// of things with identities, so a create shape leaves them out for the same
// reason a list of them is left out.
//
// Two questions that agree about everything except a map, so they are two
// functions rather than one with a flag.
func entityInside(node structure.Node) (structure.Object, bool) {
	switch shape := node.(type) {
	case structure.Object:
		return shape, shape.IsEntity()
	case structure.Reference:
		object, isObject := resolveObject(shape)
		return object, isObject && object.IsEntity()
	case structure.Sequence:
		return entityInside(shape.Element)
	case structure.Nullable:
		return entityInside(shape.Inner)
	case structure.Mapping:
		return entityInside(shape.Value)
	default:
		return structure.Object{}, false
	}
}

// resolveObject is the object a node is, following one reference.
func resolveObject(node structure.Node) (structure.Object, bool) {
	switch shape := node.(type) {
	case structure.Object:
		return shape, true
	case structure.Reference:
		if shape.Resolve == nil {
			return structure.Object{}, false
		}
		return resolveObject(shape.Resolve())
	default:
		return structure.Object{}, false
	}
}
