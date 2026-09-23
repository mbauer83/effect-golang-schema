package typescript

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

func docOf(node structure.Node) string {
	switch node := node.(type) {
	case structure.Object:
		return node.Doc
	case structure.Union:
		return node.Doc
	}
	return ""
}

func docComment(doc, prefix string) string {
	doc = strings.TrimSpace(doc)
	if doc == "" {
		return ""
	}
	lines := strings.Split(doc, "\n")
	if len(lines) == 1 {
		return prefix + "/** " + escapeComment(lines[0]) + " */\n"
	}
	var out strings.Builder
	out.WriteString(prefix + "/**\n")
	for _, line := range lines {
		out.WriteString(prefix + " * " + escapeComment(line) + "\n")
	}
	out.WriteString(prefix + " */\n")
	return out.String()
}

func escapeComment(text string) string { return strings.ReplaceAll(text, "*/", "*\\/") }

func indent(text string) string {
	lines := strings.Split(text, "\n")
	for at, line := range lines {
		if line != "" {
			lines[at] = "  " + line
		}
	}
	return strings.Join(lines, "\n")
}

var plainKey = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)

func key(name string) string {
	if plainKey.MatchString(name) {
		return name
	}
	return strconv.Quote(name)
}

func referenceName(reference structure.Reference) string {
	if reference.Name != "" || reference.Resolve == nil {
		return identifier(reference.Name)
	}
	switch target := reference.Resolve().(type) {
	case structure.Object:
		return identifier(target.Name)
	case structure.Union:
		return identifier(target.Name)
	}
	return ""
}

func identifier(name string) string {
	if at := strings.LastIndex(name, "."); at >= 0 {
		name = name[at+1:]
	}
	return name
}
