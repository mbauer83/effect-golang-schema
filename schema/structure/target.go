package structure

// What a reference identifies.

// Target is what a reference identifies: an object, by the field that is its
// identity, and what becomes of the reference when the object goes.
type Target struct {
	// Object is the target object's name, as its description declares it.
	Object string
	// Key is the target's identity field, by its declared name.
	Key string
	// OnDelete is what deleting the target does to a value that refers to it.
	OnDelete Deletion
	// Table and Column are where the target is stored, once a mapping has
	// said: empty in a domain description, which says nothing about tables.
	Table  string
	Column string
}

// Deletion is what deleting an object does to what refers to it.
type Deletion int

const (
	// Restrict refuses to delete an object something still refers to: the
	// default, because destroying data should be a decision someone states.
	Restrict Deletion = iota
	// Cascade deletes what refers to it as well.
	Cascade
	// SetNull empties the reference, which is only possible where it is
	// optional.
	SetNull
)

// WithTargets is node with every reference's target passed through resolve,
// deeply: what a mapping uses to say where each target is stored.
func WithTargets(node Node, resolve func(Target) Target) Node {
	switch node := node.(type) {
	case Scalar:
		if node.Refers != nil {
			resolved := resolve(*node.Refers)
			node.Refers = &resolved
		}
		return node
	case Object:
		fields := make([]Field, len(node.Fields))
		for index, field := range node.Fields {
			field.Node = WithTargets(field.Node, resolve)
			fields[index] = field
		}
		node.Fields = fields
		return node
	case Union:
		variants := make([]Variant, len(node.Variants))
		for index, variant := range node.Variants {
			variant.Node = WithTargets(variant.Node, resolve)
			variants[index] = variant
		}
		node.Variants = variants
		return node
	case Sequence:
		node.Element = WithTargets(node.Element, resolve)
		return node
	case Mapping:
		node.Value = WithTargets(node.Value, resolve)
		return node
	case Nullable:
		node.Inner = WithTargets(node.Inner, resolve)
		return node
	case Reference:
		if inner := node.Resolve; inner != nil {
			node.Resolve = func() Node { return WithTargets(inner(), resolve) }
		}
		return node
	default:
		return node
	}
}
