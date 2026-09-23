package typescript

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// typeOf is the TypeScript type node decodes to, written out: what Effect's
// Struct, Array, Record, NullOr and Union infer, spelled by hand for the
// components whose type cannot be inferred.
func (module *module) typeOf(node structure.Node, top bool) (string, error) {
	switch node := node.(type) {
	case structure.Scalar:
		switch node.Kind {
		case structure.Integer, structure.Number:
			return "number", nil
		case structure.Boolean:
			return "boolean", nil
		}
		return "string", nil
	case structure.Object:
		if node.Name != "" && !top {
			return identifier(node.Name), nil
		}
		return module.objectType(node.Fields, "")
	case structure.Union:
		if node.Name != "" && !top {
			return identifier(node.Name), nil
		}
		var members []string
		for _, variant := range node.Variants {
			var member string
			var err error
			if node.Discriminator != "" {
				object, _ := objectBehind(variant.Node)
				member, err = module.objectType(object.Fields,
					"readonly "+key(node.Discriminator)+": "+strconv.Quote(variant.Name)+"; ")
			} else {
				var inner string
				inner, err = module.typeOf(variant.Node, false)
				member = "{ readonly " + key(variant.Name) + ": " + inner + " }"
			}
			if err != nil {
				return "", err
			}
			members = append(members, member)
		}
		return strings.Join(members, " | "), nil
	case structure.Sequence:
		element, err := module.typeOf(node.Element, false)
		return "ReadonlyArray<" + element + ">", err
	case structure.Mapping:
		value, err := module.typeOf(node.Value, false)
		return "{ readonly [key: string]: " + value + " }", err
	case structure.Nullable:
		inner, err := module.typeOf(node.Inner, false)
		return inner + " | null", err
	case structure.Reference:
		return referenceName(node), nil
	}
	return "", fmt.Errorf("typescript: %T is not a shape this projection knows", node)
}

func (module *module) objectType(fields []structure.Field, tag string) (string, error) {
	members := []string{}
	for _, field := range fields {
		written, err := module.typeOf(field.Node, false)
		if err != nil {
			return "", err
		}
		optional := ""
		if field.Optional {
			optional = "?"
		}
		members = append(members, "readonly "+key(field.Name)+optional+": "+written+";")
	}
	return "{ " + tag + strings.Join(members, " ") + " }", nil
}
