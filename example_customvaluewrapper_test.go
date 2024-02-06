package oneof_test

import (
	"fmt"

	"github.com/dhoelle/oneof"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

func ExampleCustomValueWrapper() {
	stringerOptions := map[string]fmt.Stringer{
		"literal":     LiteralStringer(""),
		"join":        JoinStringer{},
		"exclamation": ExclamationPointsStringer(0),
	}

	cw := oneof.CustomValueWrapper{
		DiscriminatorKey: "$type",
		NestedValueKey:   "$value",
		InlineObjects:    true,
	}

	cfg := &oneof.Config{
		WrapFunc: cw.Wrap,
	}
	stringerMarshalFunc := oneof.MarshalFunc(stringerOptions, cfg)

	marshalOpts := []json.Options{
		json.WithMarshalers(stringerMarshalFunc),
		jsontext.WithIndent("  "), // make the example output easy to read
		json.Deterministic(true),  // make the example output deterministic
	}

	in := JoinStringer{
		A: JoinStringer{
			A:         LiteralStringer("Hello"),
			Separator: " ",
			B:         LiteralStringer("world"),
		},
		Separator: "",
		B:         ExclamationPointsStringer(2),
	}

	// Marshal to JSON
	b, err := json.Marshal(in, marshalOpts...)
	if err != nil {
		panic("failed to marshal: " + err.Error())
	}

	fmt.Printf("Marshaled JSON:\n%s\n", string(b))
	// Output:
	// Marshaled JSON:
	// {
	//   "$type": "join",
	//   "a": {
	//     "$type": "join",
	//     "a": {
	//       "$type": "literal",
	//       "$value": "Hello"
	//     },
	//     "b": {
	//       "$type": "literal",
	//       "$value": "world"
	//     },
	//     "separator": " "
	//   },
	//   "b": {
	//     "$type": "exclamation",
	//     "$value": 2
	//   }
	// }
}
