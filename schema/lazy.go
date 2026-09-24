package schema

// A collection an aggregate has but never holds.

import (
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// Collection is a collection an aggregate has but does not hold in its value:
// a watchlist's films, which can grow without bound. It is declared with the
// aggregate, and read and changed a statement at a time -- a page, one
// insertion, one removal -- rather than loaded whole. What Doctrine calls an
// extra lazy collection: there is no proxy, and nothing loads all of it.
//
// It holds single values: identities of another aggregate (a Ref), or plain
// values. No two of an owner's are the same.
type Collection[O, ID, E any] struct {
	owner    Schema[O]
	identity Field[O, ID]
	name     string
	element  Schema[E]
	ordered  bool
	fault    error
}

// Lazy declares a collection of owner's, by owner's identity, named name,
// holding elements described by element.
//
//	var WatchlistFilms = schema.Lazy(WatchlistSchema, WatchlistFields.ID, "films",
//		schema.Ref(catalog.FilmSchema, catalog.FilmFields.ID)).Ordered()
func Lazy[O, ID, E any](owner Schema[O], identity Field[O, ID], name string, element Schema[E]) Collection[O, ID, E] {
	declared := Collection[O, ID, E]{owner: owner, identity: identity, name: name, element: element}
	parts, object, fault := owner.projectable()
	switch {
	case fault != nil:
		declared.fault = fault
	case !isOwnIdentity(parts, identity.erased.key):
		declared.fault = fail(identity.erased.name+" is not the identity of "+object.Name, nil)
	case Validate(element) != nil:
		declared.fault = Validate(element)
	default:
		if _, single := element.node.(structure.Scalar); !single {
			declared.fault = fail("a lazy collection holds single values: an identity, or a plain value", nil)
		}
	}
	return declared
}

func isOwnIdentity[A any](parts *objectParts[A], key *fieldKey) bool {
	index := parts.index(key)
	return index >= 0 && parts.fields[index].identity
}

// Ordered is the collection in the order a person puts it in: each element has
// a place, which inserting and moving choose.
func (collection Collection[O, ID, E]) Ordered() Collection[O, ID, E] {
	collection.ordered = true
	return collection
}

// Owner is the aggregate the collection belongs to, and Identity the field
// that identifies it.
func (collection Collection[O, ID, E]) Owner() Schema[O]       { return collection.owner }
func (collection Collection[O, ID, E]) Identity() Field[O, ID] { return collection.identity }

// Name is the collection's name, Element what it holds, and IsOrdered whether
// a person orders it.
func (collection Collection[O, ID, E]) Name() string       { return collection.name }
func (collection Collection[O, ID, E]) Element() Schema[E] { return collection.element }
func (collection Collection[O, ID, E]) IsOrdered() bool    { return collection.ordered }

// Err is what makes the declaration unusable, or nil.
func (collection Collection[O, ID, E]) Err() error { return collection.fault }
