package schema

// Choices about how JSON is written and read.

import (
	"github.com/mbauer83/effect-golang-schema/schema/naming"
)

// JSONOption is a choice about how JSON is written or read.
type JSONOption func(*jsonOptions)

type jsonOptions struct {
	strategy naming.Strategy
}

// MemberNaming spells every object member, and a tagged union's tag field, by
// strategy, on the way out and on the way in. Without it, names are written and
// read as they were declared.
//
// A document read with a strategy is read strictly: a member spelled another
// way is a member this schema does not know, as it would be if it were misspelt.
func MemberNaming(strategy naming.Strategy) JSONOption {
	return func(options *jsonOptions) { options.strategy = strategy }
}

func jsonChoices(options []JSONOption) jsonOptions {
	var chosen jsonOptions
	for _, option := range options {
		option(&chosen)
	}
	return chosen
}

// NamingStrategy is how this sink spells member names.
func (sink *jsonSink) NamingStrategy() naming.Strategy { return sink.strategy }

// NamingStrategy is how this source expects member names spelled.
func (source *jsonSource) NamingStrategy() naming.Strategy { return source.strategy }
