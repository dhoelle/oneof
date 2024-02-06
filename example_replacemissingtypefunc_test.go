package oneof_test

import (
	"fmt"

	"github.com/dhoelle/oneof"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

func ExampleConfig_replaceMissingTypeFunc() {
	// Options implementing fmt.Stringer.
	// (intentionally omitting LiteralStringer)
	stringerOptions := map[string]fmt.Stringer{
		// "literal":     LiteralStringer(""),
		"join":        JoinStringer{},
		"exclamation": ExclamationPointsStringer(0),
	}

	// When oneof.MarshalFunc encounters a Go type that is not
	// in its list of options, have it generate a discriminator.
	//
	// (Note: attempting to json.Marshal the output JSON back
	// into a Go type will fail)
	cfg := &oneof.Config{
		ReplaceMissingTypeFunc: func(v any) string {
			return fmt.Sprintf("MISSING_%T", v)
		},
	}
	stringerMarshalFunc := oneof.MarshalFunc(stringerOptions, cfg)

	// Add other JSON options as desired
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
	//   "_type": "join",
	//   "_value": {
	//     "a": {
	//       "_type": "join",
	//       "_value": {
	//         "a": {
	//           "_type": "MISSING_*oneof_test.LiteralStringer",
	//           "_value": "Hello"
	//         },
	//         "b": {
	//           "_type": "MISSING_*oneof_test.LiteralStringer",
	//           "_value": "world"
	//         },
	//         "separator": " "
	//       }
	//     },
	//     "b": {
	//       "_type": "exclamation",
	//       "_value": 2
	//     }
	//   }
	// }
}
