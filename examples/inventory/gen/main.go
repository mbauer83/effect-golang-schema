// Command gen writes the inventory's Go types from its descriptions.
//
// It is a program rather than a flag on schemagen because reading a
// description means running it: a description is a Go value, so the only thing
// that can read one is Go. Fifteen lines here is a smaller price than a
// generator that shells out to the toolchain to build a harness it wrote.
package main

import (
	"fmt"
	"os"

	"github.com/mbauer83/effect-golang-schema/examples/inventory/definitions"
	"github.com/mbauer83/effect-golang-schema/schemagen"
)

func main() {
	for _, fault := range definitions.Faults() {
		fmt.Fprintf(os.Stderr, "gen: %v\n", fault)
		os.Exit(1)
	}
	written, err := schemagen.WriteBindings("inventory", definitions.Descriptions()...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen: %v\n", err)
		os.Exit(1)
	}
	// The file goes beside the descriptions, and go generate runs this with
	// that directory as its working directory -- so the name is the whole
	// path. A "../" here wrote a file one level up that nothing tracked and
	// nothing compiled, which left the drift check in CI comparing a file the
	// generator had not touched.
	if err := os.WriteFile(schemagen.BindingsFileName, written, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "gen: %v\n", err)
		os.Exit(1)
	}
}
