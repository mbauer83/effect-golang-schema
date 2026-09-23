package protobuf

// The document a description implies: a proto3 file, as data before it is text.

import (
	"strconv"
	"strings"
)

// Document is a proto3 file.
type Document struct {
	// Package is the proto package. It is the caller's, because a proto
	// package is a namespace shared with every other language reading these
	// messages and nothing in a Go description knows what it should be.
	Package string
	// Imports are the well-known types the messages use.
	Imports []string
	// Messages are every named shape reachable from the root, each declared
	// once, in the order they were first reached.
	Messages []Message
	// Services are the procedures, where the description being projected is a
	// service rather than one message. A file with none is a file of types,
	// which is what a projected message on its own is.
	Services []Service
	// Root is the name of the message the projected description became, for a
	// projection of one message. It is empty for a projection of services,
	// which have no single root.
	Root string
}

// Service is one proto3 service: a name, and the procedures it offers.
type Service struct {
	Name        string
	Description string
	Methods     []Method
}

// Method is one procedure of a service.
//
// Unary only, because that is what the description of a request and a response
// says. A streaming procedure is a different shape and would need the
// description to say which side streams.
type Method struct {
	Name        string
	Description string
	Request     string
	Response    string
}

// Message is one proto3 message.
type Message struct {
	Name        string
	Description string
	// Fields are its members. A message projected from a union has one field
	// per variant, all inside a oneof.
	Fields []Field
	// OneOf is the name of the oneof its fields belong to, or empty when they
	// are ordinary fields. A message holds at most one, because a description's
	// union is the whole of the shape it describes.
	OneOf string
}

// Field is one member of a message.
type Field struct {
	Name        string
	Description string
	Number      int
	// Type is the proto type: a scalar keyword, a message name, or a map type.
	Type string
	// Repeated says the field carries many of Type.
	Repeated bool
	// Optional says the field has explicit presence, which proto3 spells with
	// the keyword and which a oneof member may not have.
	Optional bool
	// Notes are what the description says and proto3 has no way to state --
	// the constraints, principally. They are emitted as comments, because a
	// comment is honest about not being enforced where an invented option
	// would not be.
	Notes []string
}

// Render writes the document as a .proto file.
func (document Document) Render() string {
	out := &strings.Builder{}
	out.WriteString("syntax = \"proto3\";\n")
	if document.Package != "" {
		out.WriteString("\npackage " + document.Package + ";\n")
	}
	if len(document.Imports) > 0 {
		out.WriteString("\n")
		for _, path := range document.Imports {
			out.WriteString("import \"" + path + "\";\n")
		}
	}
	for _, message := range document.Messages {
		out.WriteString("\n")
		message.render(out)
	}
	for _, service := range document.Services {
		out.WriteString("\n")
		service.render(out)
	}
	return out.String()
}

func (service Service) render(out *strings.Builder) {
	writeComment(out, "", service.Description)
	out.WriteString("service " + service.Name + " {\n")
	for _, method := range service.Methods {
		writeComment(out, "  ", method.Description)
		out.WriteString("  rpc " + method.Name +
			"(" + method.Request + ") returns (" + method.Response + ");\n")
	}
	out.WriteString("}\n")
}

func (message Message) render(out *strings.Builder) {
	writeComment(out, "", message.Description)
	out.WriteString("message " + message.Name + " {\n")

	indent := "  "
	if message.OneOf != "" {
		out.WriteString("  oneof " + message.OneOf + " {\n")
		indent = "    "
	}
	for _, field := range message.Fields {
		field.render(out, indent)
	}
	if message.OneOf != "" {
		out.WriteString("  }\n")
	}
	out.WriteString("}\n")
}

func (field Field) render(out *strings.Builder, indent string) {
	writeComment(out, indent, field.Description)
	for _, note := range field.Notes {
		out.WriteString(indent + "// " + note + "\n")
	}

	out.WriteString(indent)
	if field.Repeated {
		out.WriteString("repeated ")
	}
	if field.Optional {
		out.WriteString("optional ")
	}
	out.WriteString(field.Type + " " + field.Name + " = " +
		strconv.Itoa(field.Number) + ";\n")
}

// writeComment writes prose as a leading comment, one line per line of it, so a
// multi-paragraph doc comment does not become one unreadable line.
func writeComment(out *strings.Builder, indent string, doc string) {
	if doc == "" {
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(doc), "\n") {
		out.WriteString(indent + "// " + strings.TrimSpace(line) + "\n")
	}
}
