package schema

// What an object's constructor is given: the values its fields decoded, each
// read back through the field that decoded it.

import (
	"sort"
	"strings"
)

// Values are the decoded members of one object, as its constructor receives
// them. They are read through the fields that declared them, so each comes back
// as its own type: Of and Lookup on a Field[A, B] answer a B.
type Values struct {
	entries map[*fieldKey]slot
	// probe is set only while Object tries its constructor out, and records
	// what it reads.
	probe *probe
}

// slot is one decoded value with its type put away. The type comes back out
// through the field that put it there, which is the only field that asks for
// it by that key.
type slot interface{ isSlot() }

type typedSlot[B any] struct{ value B }

func (typedSlot[B]) isSlot() {}

// Of is the field's value among values, and its zero value when it has none:
// an optional field that was absent. A field that tells absence apart from a
// zero value is read with Lookup.
func (field Field[A, B]) Of(values Values) B {
	value, _ := field.Lookup(values)
	return value
}

// Lookup is the field's value among values, and whether it had one.
func (field Field[A, B]) Lookup(values Values) (B, bool) {
	var zero B
	if values.probe != nil {
		values.probe.read(field.erased.key, field.erased.name)
		return zero, false
	}
	entry, present := values.entries[field.erased.key]
	if !present {
		return zero, false
	}
	typed, matches := entry.(typedSlot[B])
	if !matches {
		return zero, false
	}
	return typed.value, true
}

// probe is a trial run of a constructor over no values, to find a field it
// reads that its object does not declare. Read through a field the object does
// not decode, a value would always be zero, and silently: the constructor
// would make every value as if that member were absent.
type probe struct {
	declared   map[*fieldKey]bool
	undeclared map[string]bool
}

func (probe *probe) read(key *fieldKey, name string) {
	if !probe.declared[key] {
		probe.undeclared[name] = true
	}
}

// probeConstructor runs construct once over no values and refuses it if it
// reads a field the object does not declare.
//
// A constructor given only zero values may refuse them, or panic on one; the
// refusal is ignored and the panic recovered, because only the reads matter
// here. A read after a panic goes unseen, which makes this a check that
// catches the mistake in the common case rather than a proof.
func probeConstructor[A any](construct func(Values) (A, error), fields []erasedField[A]) error {
	trial := &probe{declared: map[*fieldKey]bool{}, undeclared: map[string]bool{}}
	for _, field := range fields {
		trial.declared[field.key] = true
	}
	func() {
		defer func() { _ = recover() }()
		_, _ = construct(Values{probe: trial})
	}()
	if len(trial.undeclared) == 0 {
		return nil
	}
	names := make([]string, 0, len(trial.undeclared))
	for name := range trial.undeclared {
		names = append(names, name)
	}
	sort.Strings(names)
	return fail("the constructor reads "+strings.Join(names, ", ")+
		", which the object does not declare", nil)
}
