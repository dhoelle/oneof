package oneof

import (
	"fmt"
	"reflect"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

const (
	// The default JSON object key for type discriminators
	defaultTypeDiscriminatorKey = "_type"

	// The default JSON object key for nested values
	defaultNestedValueKey = "_value"
)

type Config struct {
	// By default, if [MarshalFunc] attempts to encode a Go type that is not in
	// the defined set of options, it returns an error.
	//
	// If ReplaceMissingTypeFunc is defined, oneof.MarshalFunc will instead call
	// ReplaceMissingTypeFunc with the unknown Go value and use the result as the
	// value for the JSON discriminator.
	ReplaceMissingTypeFunc func(any) string

	// WrapFunc wraps a type string and [jsontext.Value] in a Go type that can be
	// encoded to and decoded from JSON.
	//
	// If unset, defaults to [WrapNestObjects].
	WrapFunc func(typ string, v jsontext.Value) WrappedValue
}

func JSONOptions[T any](opts map[string]T, cfg *Config) json.Options {
	return json.JoinOptions(
		json.WithMarshalers(
			MarshalFunc(opts, cfg),
		),
		json.WithUnmarshalers(
			UnmarshalFunc(opts, cfg),
		),
	)
}

// MarshalFunc creates a [json.MarshalFuncV2] which can intercept marshaling
// behavior for values of type T and encode a JSON value that can be unmarshaled
// by [UnmarshalFunc] into a Go value of the original type T.
//
// By default, [MarshalFunc] encodes Go values into a JSON object where:
//
//   - The value of the key "_type" is a type discriminator uniquely identifying
//     the initial type T, and,
//   - The value of the key "_value" is the default JSON-encoding of the Go
//     value
//
// Encoding behavior can be customized by providing a non-nil [Config].
func MarshalFunc[T any](opts map[string]T, cfg *Config) *json.Marshalers {
	if cfg == nil {
		cfg = &Config{}
	}

	wrapFunc := cfg.WrapFunc
	if wrapFunc == nil {
		wrapFunc = WrapNested
	}

	replaceMissingTypeFunc := cfg.ReplaceMissingTypeFunc

	// Hack:
	//
	// Our strategy for generically encoding a Go type into
	// a JSON representation that includes the type is to:
	//
	//  1. Intercept the request for Marshaling T
	//  2. Marshal T in the default manner
	//  3. Create a type wrapper, which is a struct that
	//     includes the marshaled value of T as well as type
	//     information
	//  4. Marshal our type wrapper.
	//
	// However, when marshaling a Go type,
	// github.com/go-json-experiment/json looks for marshalers
	// that are registered either for that concrete type, or
	// for any interfaces which the Go type implements.
	//
	// This produces, by default, an infinite loop between
	// steps (1) and (2), where calling json.Marshal(t) will
	// re-invoke our marshalFunc
	//
	// To avoid this recursion, we set the skipNext toggle
	// below whenever we want a "default" marshaling of T. If
	// our MarshalFunc sees skipNext = true, it un-toggles
	// skipNext and returns [json.SkipFunc], which instructs
	// [json.Marshal] to skip our MarshalFunc.
	skipNext := false
	skipNextPtr := &skipNext

	marshalFunc := func(enc *jsontext.Encoder, t T, jsonopts json.Options) error {
		// If skipNextPtr is on, toggle it off and skip this
		// custom marshal function. `t` will be encoded according
		// to subsequent encoding rules; including the default
		// encoding if no other rules preempt it.
		// See [json.Marshal].
		if *skipNextPtr {
			*skipNextPtr = false
			return json.SkipFunc
		}

		// If T is an interface that captures *jsontext.Value
		// (e.g., fmt.Stringer), then our marshal func will
		// intercept attempts to marshal jsontext.Values.
		// We don't want to do that.
		if _, ok := any(t).(*jsontext.Value); ok {
			return json.SkipFunc
		}

		// Determine the discriminator value that we should
		// use for things of type `T`
		discriminatorValue, ok := discriminatorValueFor(t, opts)
		if !ok {
			if replaceMissingTypeFunc == nil {
				return ErrUnknownGoType{typ: fmt.Sprintf("%T", t)}
			}
			discriminatorValue = replaceMissingTypeFunc(t)
		}

		// Marshal t by itself
		*skipNextPtr = true // avoid recursion in Marshal below
		b, err := json.Marshal(t, jsonopts)
		if err != nil {
			return fmt.Errorf("failed to marshal t: %w", err)
		}
		jv := jsontext.Value(b)

		// Wrap the marshal'ed value with the type
		w := wrapFunc(discriminatorValue, jv)

		// Finally, marshal the wrapper
		return json.MarshalEncode(enc, w, jsonopts)
	}

	return json.MarshalFuncV2(marshalFunc)
}

// UnmarshalFunc creates a [json.UnmarshalFuncV2] which will intercept
// unmarshaling behavior for values of type T.
//
// UnmarshalFunc finds the destination Go type in its option set that matches
// the JSON type discriminator, then decodes the remaining JSON according to the
// default JSON encoding of T.
func UnmarshalFunc[T any](opts map[string]T, cfg *Config) *json.Unmarshalers {
	if cfg == nil {
		cfg = &Config{}
	}

	wrapFunc := cfg.WrapFunc
	if wrapFunc == nil {
		wrapFunc = WrapNested
	}

	// Hack:
	//
	// Our strategy for generically decoding JSON into a Go
	// type that implements T is to:
	//
	//  1. Intercept the request for Unmarshaling into T
	//  2. Unmarshal the JSON into a type wrapper, which
	//     includes type information as well as the JSON
	//     that we should use to decode into T.
	//  3. Create a new T based on the discriminator value
	//     found in (2)
	//  4. Unmarshal the "remainder" from (2) into T.
	//
	// However, when unmarshaling a Go type,
	// github.com/go-json-experiment/json looks for marshalers
	// that are registered either for that concrete type, or
	// for any interfaces which the Go type implements.
	//
	// This produces, by default, an infinite loop between
	// steps (1) and (4), where calling json.Marshal(t) will
	// re-invoke our unmarshalFunc
	//
	// To avoid this recursion, we set the skipNext toggle
	// below whenever we want a "default" unmarshaling of T.
	// If our UnmarshalFunc sees skipNext = true, it un-toggles
	// skipNext and returns [json.SkipFunc], which instructs
	// [json.Unmarshal] to skip our UnmarshalFunc.
	skipNext := false
	skipNextPtr := &skipNext

	unmarshalFunc := func(dec *jsontext.Decoder, ptr *T, jsonopts json.Options) error {
		// If skipNextPtr is on, toggle it off and skip this
		// custom unmarshal function. `t` will be decoded according
		// to subsequent decoding rules; including the default
		// encoding if no other rules preempt it.
		// See [json.Unmarshal].
		if *skipNextPtr {
			*skipNextPtr = false
			return json.SkipFunc
		}

		// We expect the JSON for this type to be wrapped in a
		// way that tells us what type of T we should decode into.
		//
		// So first, decode the input into our wrapper type.
		w := wrapFunc("", nil)
		if err := json.UnmarshalDecode(dec, &w, jsonopts); err != nil {
			return fmt.Errorf("failed to decode to type wrapper: %w", err)
		}

		// ...then, extract the type and use it to select a T
		// from our options
		opt, ok := opts[w.Type()]
		if !ok {
			return ErrUnknownDiscriminatorValue{v: w.Type()}
		}

		// ...then, unmarshal the remainder into the selected
		// option
		optPtr := &opt
		v := w.Value()

		if len(v) != 0 {
			*skipNextPtr = true // avoid recursion in Unmarshal below
			if err := json.Unmarshal(v, optPtr, jsonopts); err != nil {
				return fmt.Errorf("failed to marshal value to option type %T: %w", opt, err)
			}
		}

		*ptr = opt
		return nil
	}
	return json.UnmarshalFuncV2(unmarshalFunc)
}

// discriminatorValueFor returns the key of the first option in opts whose type,
// according to [reflect.TypeOf], matches t.
func discriminatorValueFor[T any](t T, opts map[string]T) (string, bool) {
	typeT := reflect.TypeOf(t)
	if typeT.Kind() == reflect.Ptr {
		typeT = typeT.Elem()
	}
	var key string
	for k, ot := range opts {
		typeOT := reflect.TypeOf(ot)
		if typeOT.Kind() == reflect.Ptr {
			typeOT = typeOT.Elem()
		}
		if typeOT == typeT {
			key = k
			break
		}
	}
	if key == "" {
		return "", false // not found
	}
	return key, true // found
}

// WrappedValue is the interface implemented by types that can encode a Go type
// and oneof option string into JSON, and can decode that JSON back into a
// matching Go type.
type WrappedValue interface {
	Type() string
	Value() jsontext.Value
}

// WrapNested nests the provided jsontext.Value underneath the "_value" key
// within the wrapper object
func WrapNested(typ string, v jsontext.Value) WrappedValue {
	return alwaysNestWrappedValue{
		Typ:         typ,
		NestedValue: v,
	}
}

type alwaysNestWrappedValue struct {
	Typ         string         `json:"_type"`
	NestedValue jsontext.Value `json:"_value,omitempty"`
}

func (w alwaysNestWrappedValue) Type() string { return w.Typ }
func (w alwaysNestWrappedValue) Value() jsontext.Value {
	return w.NestedValue
}

// WrapInline wraps a oneof value with default "inline" behavior, specifically:
//
//   - Type is stored under the "_type" key
//   - JSON object values are inlined into the same object as the "_type" key
//   - All non-object JSON values are nested under the "_value" key
//
// The marshaled JSON output looks like:
//
//	{
//		"my_stringers": [
//			{
//				"_type": "crypto.Hash",
//				"_value": 5
//			},
//			{
//				"_type": "url.URL",
//				"Scheme": "https",
//				"Host": "example.com"
//			},
//		]
//	}
func WrapInline(typ string, v jsontext.Value) WrappedValue {
	if v.Kind() == '{' {
		// inline objects
		return inlineObjectsWrappedValue{
			Typ:         typ,
			InlineValue: v,
		}
	}
	// nest non-objects
	return inlineObjectsWrappedValue{
		Typ:         typ,
		NestedValue: v,
	}
}

type inlineObjectsWrappedValue struct {
	Typ         string         `json:"_type"`
	NestedValue jsontext.Value `json:"_value,omitempty"`
	InlineValue jsontext.Value `json:",inline"`
}

func (w inlineObjectsWrappedValue) Type() string { return w.Typ }
func (w inlineObjectsWrappedValue) Value() jsontext.Value {
	if len(w.NestedValue) > 0 {
		return w.NestedValue
	}
	return w.InlineValue
}
