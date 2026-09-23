// Package typescript projects schemas to TypeScript: Effect Schema values and
// the types they decode to, for a frontend that reads what a Go program
// serves.
//
// A frontend that mirrors a backend's shapes by hand has one more place for the
// two to disagree, and finds out in the browser. Projected from the very
// descriptions the backend dispatches with, the two cannot drift: a field the
// backend renames is a type error in the frontend's build.
//
// The output is Effect's Schema, version 4, because a projection is only as
// useful as the decoding it gives the reader: the same values that name the
// types check what arrives, strictly, so a response the backend changed is a
// decode failure with a path in it rather than an undefined on a page.
//
// It projects the wire shape and nothing more. Scalars map to what JSON can
// carry -- text, a number, a boolean -- and constraints are the server's to
// enforce: a frontend that re-checked them would only disagree with the
// server about a rule it does not own.
package typescript

import (
	"sort"
	"strings"

	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// Module renders the named shapes reachable from roots as one TypeScript

// module: an exported Schema value and an exported type for each.

//

// Every named object and union becomes a component, in an order in which each

// is declared before it is used; a reference that would need a component not

// yet declared -- a recursive shape -- is suspended, so it is read when a value

// is decoded rather than when the module loads. Two different shapes with one

// name are refused, because one of them would silently be lost.

func Module(header string, roots ...structure.Node) (string, error) {
	module := &module{components: map[string]structure.Node{}}
	for _, root := range roots {
		if err := module.collect(root); err != nil {
			return "", err
		}
	}
	var out strings.Builder
	if header != "" {
		for _, line := range strings.Split(strings.TrimRight(header, "\n"), "\n") {
			out.WriteString("// " + line + "\n")
		}
		out.WriteString("\n")
	}
	out.WriteString("import { Schema } from \"effect\";\n")
	names := make([]string, 0, len(module.components))
	for name := range module.components {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := module.declare(name); err != nil {
			return "", err
		}
	}
	for _, declaration := range module.declarations {
		out.WriteString("\n" + declaration)
	}
	return out.String(), nil
}

type module struct {
	components   map[string]structure.Node
	declared     map[string]bool
	declaring    map[string]bool
	declarations []string
	// recursive are the components some reference had to suspend. TypeScript
	// cannot infer a type through a value that refers to itself, so these are
	// declared with the type written out and the value annotated with it,
	// which is Effect's own pattern for a recursive schema.
	recursive map[string]bool
}

// declare writes name's declaration after every component it uses.

func (module *module) declare(name string) error {
	if module.declared == nil {
		module.declared, module.declaring = map[string]bool{}, map[string]bool{}
	}
	if module.declared[name] {
		return nil
	}
	module.declaring[name] = true
	node := module.components[name]
	for _, used := range uses(node) {
		if module.declaring[used] || module.declared[used] {
			continue
		}
		if _, known := module.components[used]; known {
			if err := module.declare(used); err != nil {
				return err
			}
		}
	}
	value, err := module.expression(node, true)
	if err != nil {
		return err
	}
	delete(module.declaring, name)
	module.declared[name] = true
	doc := docComment(docOf(node), "")
	if module.recursive[name] {
		written, err := module.typeOf(node, true)
		if err != nil {
			return err
		}
		module.declarations = append(module.declarations,
			doc+"export type "+name+" = "+written+";\n"+
				"export const "+name+": Schema.Codec<"+name+"> = "+value+";\n")
		return nil
	}
	module.declarations = append(module.declarations,
		doc+"export const "+name+" = "+value+";\n"+
			"export type "+name+" = typeof "+name+".Type;\n")
	return nil
}
