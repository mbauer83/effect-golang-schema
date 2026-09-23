package structure

// What distinguishes one of a thing from another.
//
// Its own file because it is one question with one answer that several
// projections need, and because the answer has a shape people get wrong: an
// identity is a list and not a field. A disc somebody owns is identified by
// whose it is and which of theirs; an order line, by its order and its own
// number. Treating either as a single field says that half of it is unique
// across everybody, which is how one person's write comes to find, change or
// replace another's row.

// Identities are the fields that together distinguish one of these from
// another, in the order the description names them.
//
// Several, because an identity is often only unique within something else. A
// disc somebody owns is identified by whose it is and which of theirs: the
// identity its owner chose is theirs to choose, so two people may choose the
// same one and neither is the other's. A description that named only the
// second half would be saying that identity is unique across everybody, and
// a store built on it lets one person's write find, change or replace
// another's row.
//
// Empty when the description names none, which is what makes this a value
// rather than an entity.
func (object Object) Identities() []Field {
	identities := []Field{}
	for _, field := range object.Fields {
		if field.Identity {
			identities = append(identities, field)
		}
	}
	return identities
}

// Identity is the first field of the identity, and whether there is one.
//
// For the places that genuinely want one field: a generated key, the column a
// child's reference points at. A caller that means "the whole identity" wants
// Identities, and the two are different questions rather than one with a
// convenience -- which is why this says first rather than the.
func (object Object) Identity() (Field, bool) {
	named := object.Identities()
	if len(named) == 0 {
		return Field{}, false
	}
	return named[0], true
}

// IsEntity reports whether this object is a thing in its own right rather than
// a value belonging to whatever holds it.
//
// It is derived from having an identity rather than declared separately, which
// is a deliberate simplification: an object with an identity is an entity and
// one without is a value, so a separate marker could only ever agree with the
// identity or contradict it. An address inside a customer is a value and lives
// in the customer's row; an order line has an identity and lives in its own
// table.
func (object Object) IsEntity() bool {
	_, identity := object.Identity()
	return identity
}
