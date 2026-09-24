package structure

// A description as a format with a naming strategy spells it.

import (
	"github.com/mbauer83/effect-golang-schema/schema/naming"
)

// Spell is node with every object member, and every tagged union's tag
// field, spelled by strategy: the description a projection of a format with
// that strategy reads, so that a document, a TypeScript type and a table say
// the same names as the codec that writes them.
//
// Names that are data are left as they are, as the codecs leave them: a
// union's variant names, which are values saying which variant a value is, and
// the names of objects and unions, which a projection spells by its own rules.
// A reference is respelled when it is resolved, so a recursive description
// stays finite.
func Spell(node Node, strategy naming.Strategy) Node {
	if strategy.IsLiteral() {
		return node
	}
	switch node := node.(type) {
	case Object:
		fields := make([]Field, len(node.Fields))
		for index, field := range node.Fields {
			if !field.Exact {
				field.Name = strategy.Spell(field.Name)
			}
			field.Node = Spell(field.Node, strategy)
			fields[index] = field
		}
		node.Fields = fields
		return node
	case Union:
		variants := make([]Variant, len(node.Variants))
		for index, variant := range node.Variants {
			variant.Node = Spell(variant.Node, strategy)
			variants[index] = variant
		}
		node.Variants = variants
		if node.Discriminator != "" {
			node.Discriminator = strategy.Spell(node.Discriminator)
		}
		return node
	case Sequence:
		node.Element = Spell(node.Element, strategy)
		return node
	case Mapping:
		node.Value = Spell(node.Value, strategy)
		return node
	case Nullable:
		node.Inner = Spell(node.Inner, strategy)
		return node
	case Reference:
		if resolve := node.Resolve; resolve != nil {
			node.Resolve = func() Node { return Spell(resolve(), strategy) }
		}
		return node
	default:
		return node
	}
}
