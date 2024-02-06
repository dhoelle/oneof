package oneof_test

import (
	"fmt"

	"github.com/dhoelle/oneof"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

// simpleWrappedValue uses "type" and "value" as the keys for type
// discriminators and nested values, respectively. It also inlines objects.
//
// Note: if option types have fields that conflict with the type and value keys
// of the type wrapper (in this example: "type" or "value"), json.Marshal will
// not encode either field. Avoid using type and value keys which conflict with
// JSON field names used by your data types.
type simpleWrappedValue struct {
	Typ         string         `json:"type"`
	NestedValue jsontext.Value `json:"value,omitempty"`
	InlineValue jsontext.Value `json:",inline"`
}

func (w simpleWrappedValue) Type() string { return w.Typ }
func (w simpleWrappedValue) Value() jsontext.Value {
	if len(w.NestedValue) > 0 {
		return w.NestedValue
	}
	return w.InlineValue
}

func wrapSimple(typ string, v jsontext.Value) oneof.WrappedValue {
	if v.Kind() == '{' {
		// inline objects
		return simpleWrappedValue{
			Typ:         typ,
			InlineValue: v,
		}
	}
	// nest non-objects
	return simpleWrappedValue{
		Typ:         typ,
		NestedValue: v,
	}
}

func ExampleConfig_wrapFunc() {
	stringerOptions := map[string]fmt.Stringer{
		"literal":     LiteralStringer(""),
		"join":        JoinStringer{},
		"exclamation": ExclamationPointsStringer(0),
	}

	cfg := &oneof.Config{
		WrapFunc: wrapSimple,
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
	//   "type": "join",
	//   "a": {
	//     "type": "join",
	//     "a": {
	//       "type": "literal",
	//       "value": "Hello"
	//     },
	//     "b": {
	//       "type": "literal",
	//       "value": "world"
	//     },
	//     "separator": " "
	//   },
	//   "b": {
	//     "type": "exclamation",
	//     "value": 2
	//   }
	// }
}
