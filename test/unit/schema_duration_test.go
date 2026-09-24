package unit

// A duration as a whole number of units. What matters is that a value that is
// not a whole number is refused rather than rounded, and that the stored number
// keeps the width and constraints it was counted with.

import (
	"strings"
	"testing"
	"time"

	"github.com/mbauer83/effect-golang-schema/schema"
	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

var runtimeSchema = schema.Duration(time.Minute, schema.Int32().Check(schema.AtLeast[int32](0)))

func TestADurationIsWrittenAsAWholeNumberOfUnits(t *testing.T) {
	encoded, err := schema.EncodeJSON(runtimeSchema, 136*time.Minute)
	if err != nil || strings.TrimSpace(string(encoded)) != "136" {
		t.Fatalf("expected 136, got %s, %v", encoded, err)
	}
	decoded, err := schema.DecodeJSON(runtimeSchema, encoded)
	if err != nil || decoded != 136*time.Minute {
		t.Fatalf("expected the runtime back, got %v, %v", decoded, err)
	}
}

func TestADurationThatIsNotAWholeNumberOfUnitsIsRefused(t *testing.T) {
	if _, err := schema.EncodeJSON(runtimeSchema, 90*time.Second); err == nil ||
		!strings.Contains(err.Error(), "whole number") {
		t.Fatalf("expected a minute and a half refused, got %v", err)
	}
}

func TestADurationKeepsTheCountsWidthAndConstraints(t *testing.T) {
	scalar := runtimeSchema.Structure().(structure.Scalar)
	if scalar.Precision != structure.Int32Bits || len(scalar.Constraints) == 0 {
		t.Fatalf("expected the count's width and bound, got %+v", scalar)
	}
	if _, err := schema.DecodeJSON(runtimeSchema, []byte("-5")); err == nil {
		t.Fatal("expected the count's bound to refuse a negative runtime")
	}
}
