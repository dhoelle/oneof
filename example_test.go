package oneof_test

import (
	"fmt"
	"io"
	"strings"

	"github.com/dhoelle/oneof"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

//
// Some examples that implement the fmt.Stringer interface
//

type LiteralStringer string

func (s LiteralStringer) String() string { return string(s) }

type JoinStringer struct {
	A         fmt.Stringer `json:"a,omitempty"`
	B         fmt.Stringer `json:"b,omitempty"`
	Separator string       `json:"separator,omitempty"`
}

func (s JoinStringer) String() string {
	return fmt.Sprintf("%s%s%s", s.A.String(), s.Separator, s.B.String())
}

type ExclamationPointsStringer int

func (s ExclamationPointsStringer) String() string {
	return strings.Repeat("!", int(s))
}

//
// An example that implements the error interface
//

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/418
type TeapotError struct {
	ShowCode bool `json:"show_code,omitempty"`
}

func (e TeapotError) Error() string {
	if e.ShowCode {
		return "418 I'm a teapot"
	}
	return "I'm a teapot"
}

// An example struct with interface fields.
// We'll marshal and unmarshal this from JSON in the Example function, below.
type StringerAndError struct {
	Stringer fmt.Stringer `json:"stringer,omitempty"`
	Error    error        `json:"error,omitempty"`
}

func Example() {
	// Sets of options which implement the fmt.Stringer
	// and error interfaces, respectively, keyed by
	// their discriminator values
	stringerOptions := map[string]fmt.Stringer{
		"literal":     LiteralStringer(""),
		"join":        JoinStringer{},
		"exclamation": ExclamationPointsStringer(0),
	}
	errorOptions := map[string]error{
		"no_progress": io.ErrNoProgress,
		"teapot":      TeapotError{},
	}

	// Use oneof.MarshalFunc to create json.Marshalers that
	// will intercept the marshaling behavior for errors
	// and fmt.Stringers.
	//
	// Note: as of https://github.com/go-json-experiment/json/commits/2e55bd4e08b08427ba10066e9617338e1f113c53/,
	// the json v2 experiment library will redirect marshaling
	// behavior to marshal funcs that are defined on interface
	// types for both:
	//
	//   a) interface values
	//      (e.g., `var err error = TeapotError{}`), and,
	//   b) types which satisfy an interface
	//      (e.g., `err := TeapotError{}`, where type
	//      TeapotError satisfies interface error`)
	//
	errorMarshalFunc := oneof.MarshalFunc(errorOptions, nil)
	stringerMarshalFunc := oneof.MarshalFunc(stringerOptions, nil)

	// Combine our marshal funcs into a single *json.Marshalers
	marshalers := json.NewMarshalers(
		errorMarshalFunc,
		stringerMarshalFunc,
	)

	// Add other JSON options as desired
	marshalOpts := []json.Options{
		json.WithMarshalers(marshalers),
		jsontext.WithIndent("  "), // make the example output easy to read
		json.Deterministic(true),  // make the example output deterministic
	}

	in := StringerAndError{
		Error: TeapotError{ShowCode: true},
		Stringer: JoinStringer{
			A: JoinStringer{
				A:         LiteralStringer("Hello"),
				Separator: " ",
				B:         LiteralStringer("world"),
			},
			Separator: "",
			B:         ExclamationPointsStringer(2),
		},
	}

	// Marshal to JSON
	b, err := json.Marshal(in, marshalOpts...)
	if err != nil {
		panic("failed to marshal: " + err.Error())
	}

	// Build unmarshal funcs, similar to the process above
	errorUnmarshalFunc := oneof.UnmarshalFunc(errorOptions, nil)
	stringerUnmarshalFunc := oneof.UnmarshalFunc(stringerOptions, nil)
	unmarshalers := json.NewUnmarshalers(
		errorUnmarshalFunc,
		stringerUnmarshalFunc,
	)
	unmarshalOpts := []json.Options{
		json.WithUnmarshalers(unmarshalers),
	}

	// Unmarshal our JSON into a new, empty StringerAndError
	out := StringerAndError{}
	if err := json.Unmarshal(b, &out, unmarshalOpts...); err != nil {
		panic("failed to unmarshal: " + err.Error())
	}

	fmt.Printf("Marshaled JSON:\n")
	fmt.Printf("%s\n", string(b))
	fmt.Printf("\n")
	fmt.Printf("Output from unmarshaled Go values:\n")
	fmt.Printf("  error: %s\n", out.Error.Error())
	fmt.Printf("  string: %s\n", out.Stringer.String())

	// Output:
	// Marshaled JSON:
	// {
	//   "stringer": {
	//     "_type": "join",
	//     "_value": {
	//       "a": {
	//         "_type": "join",
	//         "_value": {
	//           "a": {
	//             "_type": "literal",
	//             "_value": "Hello"
	//           },
	//           "b": {
	//             "_type": "literal",
	//             "_value": "world"
	//           },
	//           "separator": " "
	//         }
	//       },
	//       "b": {
	//         "_type": "exclamation",
	//         "_value": 2
	//       }
	//     }
	//   },
	//   "error": {
	//     "_type": "teapot",
	//     "_value": {
	//       "show_code": true
	//     }
	//   }
	// }
	//
	// Output from unmarshaled Go values:
	//   error: 418 I'm a teapot
	//   string: Hello world!!
}
