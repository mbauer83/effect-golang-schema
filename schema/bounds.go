package schema

// The constraints there are.
//
// One constructor each rather than one generic narrowing taking a measuring
// function, because measuring a value is type-dependent -- a number is
// compared, a string is counted, a list is counted differently -- and a
// generic one would have to take a function nobody wants to write. What they
// have in common is the type they are about, which is what keeps a length off
// a number.

import (
	"regexp"
	"strconv"

	"github.com/mbauer83/effect-golang-schema/schema/structure"
)

// numeric is every Go type a bound can be stated over.
//
// It is the whole numeric breadth of the language rather than the two shapes
// the wire has, which is what makes a bound outside a type's range a compile
// error: AtMost[int8](200) does not build, because 200 is not an int8.
type numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// AtLeast admits values no less than minimum.
func AtLeast[A numeric](minimum A) Constraint[A] {
	return narrowing(structure.AtLeast{Value: float64(minimum)},
		func(value A) error {
			if value < minimum {
				return fail("is less than "+number(minimum), nil)
			}
			return nil
		})
}

// AtMost admits values no greater than maximum.
func AtMost[A numeric](maximum A) Constraint[A] {
	return narrowing(structure.AtMost{Value: float64(maximum)},
		func(value A) error {
			if value > maximum {
				return fail("is greater than "+number(maximum), nil)
			}
			return nil
		})
}

// Above admits values strictly greater than the bound.
func Above[A numeric](bound A) Constraint[A] {
	return narrowing(structure.Above{Value: float64(bound)},
		func(value A) error {
			if value <= bound {
				return fail("is not above "+number(bound), nil)
			}
			return nil
		})
}

// Below admits values strictly less than the bound.
func Below[A numeric](bound A) Constraint[A] {
	return narrowing(structure.Below{Value: float64(bound)},
		func(value A) error {
			if value >= bound {
				return fail("is not below "+number(bound), nil)
			}
			return nil
		})
}

// MinLength admits strings of at least the given length, counted in characters
// rather than bytes: a length a client can check is the one it can see.
func MinLength(atLeast int) Constraint[string] {
	return narrowing(structure.MinLength{Value: atLeast},
		func(value string) error {
			if length(value) < atLeast {
				return fail("is shorter than "+strconv.Itoa(atLeast)+" characters", nil)
			}
			return nil
		})
}

// MaxLength admits strings of at most the given length, in characters.
func MaxLength(atMost int) Constraint[string] {
	return narrowing(structure.MaxLength{Value: atMost},
		func(value string) error {
			if length(value) > atMost {
				return fail("is longer than "+strconv.Itoa(atMost)+" characters", nil)
			}
			return nil
		})
}

// Matching admits strings the expression matches. The syntax is Go's, which is
// RE2; a pattern that does not compile is a declaration mistake and is
// reported by Validate rather than panicking at the first request.
func Matching(expression string) Constraint[string] {
	compiled, err := regexp.Compile(expression)
	if err != nil {
		return Constraint[string]{fault: fail("the pattern does not compile", err)}
	}
	return narrowing(structure.Pattern{Expression: expression},
		func(value string) error {
			if !compiled.MatchString(value) {
				return fail("does not match "+expression, nil)
			}
			return nil
		})
}

// MinItems admits sequences of at least the given length.
//
// The element type has to be named -- MinItems[string](1) -- because there is
// no longer an inner schema to read it from, and Go does not infer a type
// argument from the parameter a value is passed as.
func MinItems[A any](atLeast int) Constraint[[]A] {
	return narrowing(structure.MinItems{Value: atLeast},
		func(value []A) error {
			if len(value) < atLeast {
				return fail("has fewer than "+strconv.Itoa(atLeast)+" items", nil)
			}
			return nil
		})
}

// MaxItems admits sequences of at most the given length.
func MaxItems[A any](atMost int) Constraint[[]A] {
	return narrowing(structure.MaxItems{Value: atMost},
		func(value []A) error {
			if len(value) > atMost {
				return fail("has more than "+strconv.Itoa(atMost)+" items", nil)
			}
			return nil
		})
}
