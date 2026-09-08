package architecture

// Nothing here holds a value it cannot name, and here there are no exceptions.
//
// The other modules each have a file or two where something outside them is
// untyped and something has to meet it -- a driver's values, a broker's
// headers, a codec's messages. This module meets nothing: a description is
// data and the formats it projects to are specifications, so there is no
// boundary to cross and no exemption to record.

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestNoDescriptionEscapesIntoATopType(t *testing.T) {
	typeParameters := regexp.MustCompile(`\[[\w,\s]*any[\w,\s]*\]`)
	topType := regexp.MustCompile(`\binterface\{\}|\bany\b`)

	walk := func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		for number, line := range strings.Split(readSource(t, path), "\n") {
			code, _, _ := strings.Cut(line, "//")
			code = typeParameters.ReplaceAllString(code, "")
			if topType.MatchString(code) {
				t.Errorf("%s:%d uses a top type: %s",
					display(t, path), number+1, strings.TrimSpace(line))
			}
		}
		return nil
	}
	if err := filepath.WalkDir(moduleRoot(t), walk); err != nil {
		t.Fatal(err)
	}
}
