package schema

// An object's member names as each format spells them.

import (
	"sync"

	"github.com/mbauer83/effect-golang-schema/schema/naming"
)

// NamingFormat is a format whose members are named by a strategy: JSON in
// camelCase, a table in snake_case. A Sink or Source implements it to have every
// object member, and a tagged union's tag field, spelled that way. One that
// does not implement it writes and reads names as they were declared.
//
// Names that are data are never respelled: a map's keys, and a union's variant
// names, which are values that say which variant a value is.
type NamingFormat interface {
	NamingStrategy() naming.Strategy
}

func sinkStrategy(into Sink) naming.Strategy {
	if format, named := into.(NamingFormat); named {
		return format.NamingStrategy()
	}
	return naming.Literal
}

func sourceStrategy(from Source) naming.Strategy {
	if format, named := from.(NamingFormat); named {
		return format.NamingStrategy()
	}
	return naming.Literal
}

// fieldSpelling are an object's fields under one strategy: each field's name
// in declaration order, the fields by name for a decoder, and the names a
// decoder requires.
type fieldSpelling[A any] struct {
	names    []string
	byName   map[string]erasedField[A]
	required []string
	// fault is two fields this strategy spells alike, which is refused when a
	// value is written or read with it.
	fault error
}

// memberNames spell an object's fields once per strategy, the first time a
// format with that strategy meets the object, rather than once per value.
type memberNames[A any] struct {
	fields  []erasedField[A]
	literal *fieldSpelling[A]
	// spelled holds a *spelledFields[A] per strategy name.
	byStrategy sync.Map
}

func newMemberNames[A any](fields []erasedField[A]) *memberNames[A] {
	return &memberNames[A]{fields: fields, literal: spellFields(fields, naming.Literal)}
}

func (names *memberNames[A]) under(strategy naming.Strategy) *fieldSpelling[A] {
	if strategy.IsLiteral() {
		return names.literal
	}
	if known, found := names.byStrategy.Load(strategy.Name()); found {
		return known.(*fieldSpelling[A])
	}
	spelling, _ := names.byStrategy.LoadOrStore(strategy.Name(), spellFields(names.fields, strategy))
	return spelling.(*fieldSpelling[A])
}

func spellFields[A any](fields []erasedField[A], strategy naming.Strategy) *fieldSpelling[A] {
	spelling := &fieldSpelling[A]{
		names:  make([]string, len(fields)),
		byName: make(map[string]erasedField[A], len(fields)),
	}
	for index, field := range fields {
		name := field.name
		if !field.literalName {
			name = strategy.Spell(field.name)
		}
		if earlier, taken := spelling.byName[name]; taken && spelling.fault == nil {
			spelling.fault = fail("fields "+earlier.name+" and "+field.name+
				" are both "+name+" in "+strategy.Name(), nil)
		}
		spelling.names[index] = name
		spelling.byName[name] = field
		if !field.optional {
			spelling.required = append(spelling.required, name)
		}
	}
	return spelling
}

// nameSpelling is one name that is a member name, as each strategy spells it,
// spelled once per strategy: a tagged union's tag field.
type nameSpelling struct {
	name string
	// spelled holds the name as a string per strategy name.
	byStrategy sync.Map
}

func (name *nameSpelling) under(strategy naming.Strategy) string {
	if strategy.IsLiteral() {
		return name.name
	}
	if known, found := name.byStrategy.Load(strategy.Name()); found {
		return known.(string)
	}
	spelling, _ := name.byStrategy.LoadOrStore(strategy.Name(), strategy.Spell(name.name))
	return spelling.(string)
}
