package schema

// A length of time as a whole number of units.

import (
	"math"
	"time"
)

// counting is the Go integer types a duration can be counted in.
type counting interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint8 | ~uint16 | ~uint32
}

// Duration is a length of time written as a whole number of units -- seconds,
// minutes -- counted by count, whose width and constraints are the stored
// number's: a runtime is Duration(time.Minute, Int32().Check(AtLeast[int32](0))).
//
// A duration that is not a whole number of units is refused rather than
// rounded, and so is a count that does not fit, because a value written one
// way and read back another is a value lost. Name the field for its unit --
// runtimeMinutes -- since the number alone does not say it.
func Duration[C counting](unit time.Duration, count Schema[C]) Schema[time.Duration] {
	if unit <= 0 {
		return faultySchema[time.Duration](count.node,
			fail("a duration is counted in a positive length of time", nil))
	}
	return TransformOrFail(count,
		func(units C) (time.Duration, error) {
			held := int64(units)
			if held > math.MaxInt64/int64(unit) || held < math.MinInt64/int64(unit) {
				return 0, fail("is more than a duration can hold", nil)
			}
			return time.Duration(held) * unit, nil
		},
		func(length time.Duration) (C, error) {
			if length%unit != 0 {
				return 0, fail(length.String()+" is not a whole number of "+unit.String(), nil)
			}
			units := int64(length / unit)
			if int64(C(units)) != units {
				return 0, fail(length.String()+" is more than this count can hold", nil)
			}
			return C(units), nil
		})
}
