package typescript

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

func (module *module) expression(node structure.Node, top bool) (string, error) {
	switch node := node.(type) {
	case structure.Scalar:
		return scalar(node.Kind), nil
	case structure.Object:
		if node.Name != "" && !top {
			return module.reference(identifier(node.Name)), nil
		}
		return module.object(node.Fields, "")
	case structure.Union:
		if node.Name != "" && !top {
			return module.reference(identifier(node.Name)), nil
		}
		return module.union(node)
	case structure.Sequence:
		element, err := module.expression(node.Element, false)
		return "Schema.Array(" + element + ")", err
	case structure.Mapping:
		value, err := module.expression(node.Value, false)
		return "Schema.Record(Schema.String, " + value + ")", err
	case structure.Nullable:
		inner, err := module.expression(node.Inner, false)
		return "Schema.NullOr(" + inner + ")", err
	case structure.Reference:
		return module.reference(referenceName(node)), nil
	}
	return "", fmt.Errorf("typescript: %T is not a shape this projection knows", node)
}

func (module *module) reference(name string) string {
	if module.declared[name] {
		return name
	}
	if module.recursive == nil {
		module.recursive = map[string]bool{}
	}
	module.recursive[name] = true
	return "Schema.suspend((): Schema.Codec<" + name + "> => " + name + ")"
}

func (module *module) object(fields []structure.Field, tag string) (string, error) {
	var members []string
	if tag != "" {
		members = append(members, tag)
	}
	for _, field := range fields {
		value, err := module.expression(field.Node, false)
		if err != nil {
			return "", err
		}
		if field.Optional {
			value = "Schema.optionalKey(" + value + ")"
		}
		members = append(members, docComment(field.Description, "  ")+"  "+key(field.Name)+": "+value+",")
	}
	if len(members) == 0 {
		return "Schema.Struct({})", nil
	}
	return "Schema.Struct({\n" + strings.Join(members, "\n") + "\n})", nil
}

func (module *module) union(union structure.Union) (string, error) {
	var members []string
	for _, variant := range union.Variants {
		var member string
		var err error
		if union.Discriminator != "" {
			object, ok := objectBehind(variant.Node)
			if !ok {
				return "", fmt.Errorf("typescript: variant %s of %s is told apart by %s but is not an object",
					variant.Name, union.Name, union.Discriminator)
			}
			tag := "  " + key(union.Discriminator) + ": Schema.Literal(" + strconv.Quote(variant.Name) + "),"
			member, err = module.object(object.Fields, tag)
		} else {
			var inner string
			inner, err = module.expression(variant.Node, false)
			member = "Schema.Struct({ " + key(variant.Name) + ": " + inner + " })"
		}
		if err != nil {
			return "", err
		}
		members = append(members, member)
	}
	return "Schema.Union([\n" + indent(strings.Join(members, ",\n")) + ",\n])", nil
}

func objectBehind(node structure.Node) (structure.Object, bool) {
	switch node := node.(type) {
	case structure.Object:
		return node, true
	case structure.Reference:
		if node.Resolve != nil {
			return objectBehind(node.Resolve())
		}
	}
	return structure.Object{}, false
}

func scalar(kind structure.Kind) string {
	switch kind {
	case structure.Integer, structure.Number:
		return "Schema.Number"
	case structure.Boolean:
		return "Schema.Boolean"
	default:
		// Text, bytes and timestamps all arrive as JSON text.
		return "Schema.String"
	}
}
