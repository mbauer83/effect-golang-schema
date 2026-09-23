package typescript

import (
	"fmt"

	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// collect finds every named shape under node.

func (module *module) collect(node structure.Node) error {
	switch node := node.(type) {
	case structure.Object:
		if node.Name != "" {
			if seen, ok := module.components[identifier(node.Name)]; ok {
				if sameName(seen, node) {
					return nil
				}
				return fmt.Errorf("typescript: two shapes are both named %s", identifier(node.Name))
			}
			module.components[identifier(node.Name)] = node
		}
		for _, field := range node.Fields {
			if err := module.collect(field.Node); err != nil {
				return err
			}
		}
	case structure.Union:
		if node.Name != "" {
			if _, ok := module.components[identifier(node.Name)]; ok {
				return nil
			}
			module.components[identifier(node.Name)] = node
		}
		for _, variant := range node.Variants {
			if err := module.collect(variant.Node); err != nil {
				return err
			}
		}
	case structure.Sequence:
		return module.collect(node.Element)
	case structure.Mapping:
		return module.collect(node.Value)
	case structure.Nullable:
		return module.collect(node.Inner)
	case structure.Reference:
		if _, ok := module.components[referenceName(node)]; ok || node.Resolve == nil {
			return nil
		}
		return module.collect(node.Resolve())
	}
	return nil
}

// sameName reports whether seen is node reached again by another path rather

// than a different shape that happens to share its name: the same fields,

// under the same names, in the same order.

func sameName(seen structure.Node, node structure.Object) bool {
	object, ok := seen.(structure.Object)
	if !ok || len(object.Fields) != len(node.Fields) {
		return false
	}
	for at, field := range object.Fields {
		if field.Name != node.Fields[at].Name {
			return false
		}
	}
	return true
}

// uses are the components node names, not looking inside them.

func uses(node structure.Node) []string {
	var names []string
	var walk func(node structure.Node, top bool)
	walk = func(node structure.Node, top bool) {
		switch node := node.(type) {
		case structure.Object:
			if node.Name != "" && !top {
				names = append(names, identifier(node.Name))
				return
			}
			for _, field := range node.Fields {
				walk(field.Node, false)
			}
		case structure.Union:
			if node.Name != "" && !top {
				names = append(names, identifier(node.Name))
				return
			}
			for _, variant := range node.Variants {
				walk(variant.Node, false)
			}
		case structure.Sequence:
			walk(node.Element, false)
		case structure.Mapping:
			walk(node.Value, false)
		case structure.Nullable:
			walk(node.Inner, false)
		case structure.Reference:
			names = append(names, referenceName(node))
		}
	}
	walk(node, true)
	return names
}

// expression is the Schema value for node. top is true for the node a

// component is declared as, which is spelled out rather than referred to.

// reference names a component, suspended when it is not declared yet.

// typeOf is the TypeScript type node decodes to, written out: what Effect's

// Struct, Array, Record, NullOr and Union infer, spelled by hand for the

// components whose type cannot be inferred.

// object is a Struct of fields, with tag first when the object is a variant

// of a union told apart by a field of its own.

// union is either of the two wire forms a union has: the variant's name as a

// field inside its own object, or as the single member of a wrapping one.

// key is a member name, quoted when it is not an identifier.

// referenceName is the component a reference points at: its own name when it

// has one, and otherwise the name of the shape it resolves to, which is how a

// schema suspended for recursion refers to itself.

// identifier is the TypeScript name of a component: the last segment of its

// schema name, which a qualified name like "catalog.Film" carries after its

// dot.
