// Package naming is how a name written once is spelled by each format that
// carries it: a naming strategy, as Quill and zio-json call it.
//
// A field is named once, in whatever case its author wrote. A format states its
// strategy -- JSON in camelCase, SQL in snake_case -- and every name it carries
// is spelled that way. So the same member is `syncedAt` on the wire and
// `synced_at` in a table without being written twice, and a name that differs
// between them is a deliberate override rather than the default.
//
// A strategy works on words. Words splits a name at its separators and at its
// case changes, so `syncedAt`, `SyncedAt`, `synced_at` and `synced-at` are all
// the words synced and at, and any of them can be respelled in any strategy.
package naming

import (
	"strings"
	"unicode"
)

// Strategy is how a format spells a name. Its zero value is Literal.
type Strategy struct {
	name string
	join func(words []string) string
}

var (
	// Literal spells a name exactly as it was written, which is what every
	// format did before strategies existed and what one that states none does.
	Literal = Strategy{}
	// CamelCase is `syncedAt`: JSON's and JavaScript's convention.
	CamelCase = Strategy{name: "camelCase", join: func(words []string) string {
		return strings.Join(words[:1], "") + joinTitled(words[1:], "")
	}}
	// PascalCase is `SyncedAt`: a type's name in most languages.
	PascalCase = Strategy{name: "PascalCase", join: func(words []string) string {
		return joinTitled(words, "")
	}}
	// SnakeCase is `synced_at`: SQL's and protobuf's convention.
	SnakeCase = Strategy{name: "snake_case", join: func(words []string) string {
		return strings.Join(words, "_")
	}}
	// KebabCase is `synced-at`: a URL's or a header's.
	KebabCase = Strategy{name: "kebab-case", join: func(words []string) string {
		return strings.Join(words, "-")
	}}
)

// Custom is a strategy of the program's own: a name for it, which is how two
// uses of it are known to be the same one, and how it joins lower-case words.
func Custom(name string, join func(words []string) string) Strategy {
	return Strategy{name: name, join: join}
}

// Name is the strategy's name, and empty for Literal.
func (strategy Strategy) Name() string { return strategy.name }

// IsLiteral reports whether names are spelled as they were written.
func (strategy Strategy) IsLiteral() bool { return strategy.join == nil }

// Spell is name as this strategy spells it.
func (strategy Strategy) Spell(name string) string {
	if strategy.join == nil {
		return name
	}
	words := Words(name)
	if len(words) == 0 {
		return name
	}
	return strategy.join(words)
}

// Words are the lower-case words a name is made of.
//
// A name splits at an underscore, a hyphen, a space or a dot, and where its
// case changes: before an upper-case letter that follows a lower-case letter or
// a digit (`syncedAt`), and before the last upper-case letter of a run that a
// lower-case letter follows (`HTTPServer` is http and server). Digits stay with
// the word before them, so `w500` and `h264Profile` keep their numbers.
func Words(name string) []string {
	var words []string
	var word []rune
	runes := []rune(name)
	flush := func() {
		if len(word) > 0 {
			words = append(words, strings.ToLower(string(word)))
			word = word[:0]
		}
	}
	for index, r := range runes {
		switch {
		case r == '_' || r == '-' || r == ' ' || r == '.':
			flush()
			continue
		case unicode.IsUpper(r) && index > 0:
			previous := runes[index-1]
			nextIsLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
			if unicode.IsLower(previous) || unicode.IsDigit(previous) ||
				(unicode.IsUpper(previous) && nextIsLower) {
				flush()
			}
		}
		word = append(word, r)
	}
	flush()
	return words
}

func joinTitled(words []string, separator string) string {
	titled := make([]string, len(words))
	for index, word := range words {
		runes := []rune(word)
		runes[0] = unicode.ToUpper(runes[0])
		titled[index] = string(runes)
	}
	return strings.Join(titled, separator)
}
