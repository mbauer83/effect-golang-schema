package jsonschema

import (
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// Project turns a schema's structure into a JSON Schema document.
//
// A named object or union becomes a component and is referred to wherever it
// appears, so a type used by ten endpoints is described once. That is also what
// makes a recursive type terminate: the second time a name is reached, a
// reference is emitted rather than the shape again.
//
// Projection is total. Every structure this module can build has a rendering,
// including one that carries no documentation at all.
func Project(node structure.Node) Document {
	projection := &projector{components: map[string]Node{}, ancestors: map[string]bool{}}
	root := projection.node(node)
	return Document{Root: root, Components: projection.components}
}

// ProjectAll projects several structures against one component set, which is
// what an OpenAPI document needs: every endpoint's shapes share the components.
func ProjectAll(nodes ...structure.Node) ([]Node, map[string]Node) {
	return ProjectAllWithPointer(ReferenceTo, nodes...)
}

// ProjectAllWithPointer is ProjectAll with the caller's own pointer form.
//
// A standalone schema keeps its components under $defs; an OpenAPI document
// keeps them under #/components/schemas. The shapes are identical, so only the
// pointer differs and only the enclosing document knows what it should be.
func ProjectAllWithPointer(
	pointer func(string) string,
	nodes ...structure.Node,
) ([]Node, map[string]Node) {
	projection := &projector{
		components: map[string]Node{},
		ancestors:  map[string]bool{},
		reference:  pointer,
	}
	schemas := make([]Node, 0, len(nodes))
	for _, node := range nodes {
		schemas = append(schemas, projection.node(node))
	}
	return schemas, projection.components
}

// ReferenceTo names a component the way the 2020-12 dialect does.
func ReferenceTo(name string) string {
	return "#/$defs/" + name
}

type projector struct {
	components map[string]Node
	// ancestors guards against a recursive type: a name reached while it is
	// still being described is referred to rather than expanded again.
	ancestors map[string]bool
	// reference builds the pointer for a component name, so an OpenAPI
	// projection can point at its own components section instead of $defs.
	reference func(string) string
}

func (projection *projector) pointer(name string) string {
	if projection.reference != nil {
		return projection.reference(name)
	}
	return ReferenceTo(name)
}

func (projection *projector) node(node structure.Node) Node {
	switch shape := node.(type) {
	case structure.Scalar:
		return scalarNode(shape)
	case structure.Object:
		return projection.object(shape)
	case structure.Sequence:
		element := projection.node(shape.Element)
		return Node{Type: "array", Items: &element, Bounds: bounds(shape.Constraints)}
	case structure.Mapping:
		value := projection.node(shape.Value)
		return Node{Type: "object", Values: &value}
	case structure.Union:
		return projection.union(shape)
	case structure.Nullable:
		// A choice between the shape and null, rather than a widened type
		// keyword, because the inner shape may be a reference -- and a $ref
		// has no type keyword to widen.
		return Node{OneOf: []Node{projection.node(shape.Inner), {Type: "null"}}}
	case structure.Reference:
		return projection.referenceNode(shape)
	default:
		// The structure set is sealed, so this is unreachable unless a node was
		// added without a projection for it. Saying so beats rendering nothing.
		return Node{Description: "unprojectable structure"}
	}
}

func (projection *projector) object(shape structure.Object) Node {
	if shape.Name == "" {
		return projection.inlineObject(shape)
	}
	if projection.ancestors[shape.Name] {
		return Node{Ref: projection.pointer(shape.Name)}
	}
	if _, described := projection.components[shape.Name]; described {
		return Node{Ref: projection.pointer(shape.Name)}
	}

	projection.ancestors[shape.Name] = true
	schema := projection.inlineObject(shape)
	delete(projection.ancestors, shape.Name)
	projection.components[shape.Name] = schema
	return Node{Ref: projection.pointer(shape.Name)}
}

func (projection *projector) inlineObject(shape structure.Object) Node {
	schema := Node{Type: "object", Description: shape.Description}
	for _, field := range shape.Fields {
		member := projection.node(field.Node)
		if field.Description != "" {
			member.Description = field.Description
		}
		schema.Properties = append(schema.Properties, Property{Name: field.Name, Schema: member})
		if !field.Optional {
			schema.Required = append(schema.Required, field.Name)
		}
	}
	return schema
}

func (projection *projector) union(shape structure.Union) Node {
	describe := func() Node {
		if shape.Discriminator != "" {
			return projection.taggedUnion(shape)
		}
		schema := Node{Description: shape.Description}
		for _, variant := range shape.Variants {
			// A union names the chosen variant as the single member of an
			// object, so an alternative projects as that object rather than as
			// the variant's own shape. The projection has to agree with the
			// codec; a document describing the unwrapped shape would describe
			// something this module never writes.
			schema.OneOf = append(schema.OneOf, Node{
				Type:        "object",
				Description: variant.Description,
				Required:    []string{variant.Name},
				Properties: []Property{{
					Name:   variant.Name,
					Schema: projection.node(variant.Node),
				}},
			})
		}
		return schema
	}

	if shape.Name == "" {
		return describe()
	}
	if projection.ancestors[shape.Name] {
		return Node{Ref: projection.pointer(shape.Name)}
	}
	if _, described := projection.components[shape.Name]; described {
		return Node{Ref: projection.pointer(shape.Name)}
	}

	projection.ancestors[shape.Name] = true
	schema := describe()
	delete(projection.ancestors, shape.Name)
	projection.components[shape.Name] = schema
	return Node{Ref: projection.pointer(shape.Name)}
}

// taggedUnion describes a union whose variants are told apart by a field.
//
// Each alternative is the variant's own shape and the field that names it,
// which is what allOf is for: the variant may be a component, and a reference
// has nothing to add a property to. The const in each is what validates; the
// discriminator beside them is an annotation for a reader that understands one.
func (projection *projector) taggedUnion(shape structure.Union) Node {
	schema := Node{Description: shape.Description, Discriminator: shape.Discriminator}
	for _, variant := range shape.Variants {
		tag := Node{
			Type:     "object",
			Required: []string{shape.Discriminator},
			Properties: []Property{{
				Name:   shape.Discriminator,
				Schema: Node{Type: "string", Const: variant.Name},
			}},
		}
		schema.OneOf = append(schema.OneOf, Node{
			Description: variant.Description,
			AllOf:       []Node{projection.node(variant.Node), tag},
		})
	}
	return schema
}

func (projection *projector) referenceNode(shape structure.Reference) Node {
	if shape.Name != "" {
		return Node{Ref: projection.pointer(shape.Name)}
	}
	if shape.Resolve == nil {
		return Node{Description: "unresolvable reference"}
	}
	return projection.node(shape.Resolve())
}
