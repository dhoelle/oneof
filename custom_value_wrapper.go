package oneof

import (
	"fmt"
	"sort"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

// CustomValueWrapper can be used to quickly build a custom WrapFunc.
//
// Note: CustomValueWrapper does additional work at runtime to dynamically
// encode and decode JSON. In many cases, this will add additional operations
// over standard value wrappers, including additional round trips of
// json.Marshal + json.Unmarshal, and additional iterations over the input
// value.
//
// If performance is a concern, consider building a custom [WrappedValue] and
// [WrapFunc]. See [Config] for an example (WrapFunc).
type CustomValueWrapper struct {
	DiscriminatorKey string
	NestedValueKey   string
	InlineObjects    bool
}

func (b CustomValueWrapper) Empty() WrappedValue {
	discriminatorKey := defaultTypeDiscriminatorKey
	if b.DiscriminatorKey != "" {
		discriminatorKey = b.DiscriminatorKey
	}

	nestedValueKey := defaultNestedValueKey
	if b.NestedValueKey != "" {
		nestedValueKey = b.NestedValueKey
	}

	return customKeyWrappedType{
		discriminatorKey: discriminatorKey,
		nestedValueKey:   nestedValueKey,
	}
}

func (b CustomValueWrapper) Wrap(typ string, v jsontext.Value) WrappedValue {
	discriminatorKey := defaultTypeDiscriminatorKey
	if b.DiscriminatorKey != "" {
		discriminatorKey = b.DiscriminatorKey
	}

	nestedValueKey := defaultNestedValueKey
	if b.NestedValueKey != "" {
		nestedValueKey = b.NestedValueKey
	}

	t := customKeyWrappedType{
		discriminatorKey:   discriminatorKey,
		discriminatorValue: typ,
		nestedValueKey:     nestedValueKey,
	}

	if v.Kind() == '{' && b.InlineObjects {
		t.inlineValue = v
	} else {
		t.nestedValue = v
	}

	return t
}

type customKeyWrappedType struct {
	discriminatorKey   string
	discriminatorValue string
	nestedValueKey     string
	nestedValue        jsontext.Value
	inlineValue        jsontext.Value
}

func (w customKeyWrappedType) Type() string { return w.discriminatorValue }
func (w customKeyWrappedType) Value() jsontext.Value {
	if len(w.inlineValue) > 0 {
		return w.inlineValue
	}
	return w.nestedValue
}

func (w customKeyWrappedType) MarshalJSONV2(enc *jsontext.Encoder, opts json.Options) error {
	if w.discriminatorKey == "" {
		return fmt.Errorf("custom discriminator key is empty")
	}

	if err := enc.WriteToken(jsontext.ObjectStart); err != nil {
		return fmt.Errorf("failed to write object start token: %w", err)
	}
	if err := enc.WriteToken(jsontext.String(w.discriminatorKey)); err != nil {
		return fmt.Errorf("failed to write discriminator key token %s: %w", w.discriminatorKey, err)
	}
	if err := enc.WriteToken(jsontext.String(w.discriminatorValue)); err != nil {
		return fmt.Errorf("failed to write discriminator value token %s: %w", w.discriminatorValue, err)
	}

	switch {

	case len(w.inlineValue) > 0 && len(w.nestedValue) > 0:
		return fmt.Errorf("found both inline and nested values")

	case len(w.inlineValue) > 0:
		// Manually inline the values by marshalling to a
		// map[string]jsontext.Value, then iterating through the
		// map and writing each key+value pair
		m := map[string]jsontext.Value{}
		if err := json.Unmarshal(w.inlineValue, &m, opts); err != nil {
			return fmt.Errorf("failed to unmarshal inline value to map[string]jsontext.Value: %w", err)
		}

		deterministic, ok := json.GetOption(opts, json.Deterministic)
		switch {
		case ok && deterministic:
			// json.Deterministic is set, so iterate through map
			// keys in lexicographical order
			var keys []string

			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				v := m[k]
				if err := enc.WriteToken(jsontext.String(k)); err != nil {
					return fmt.Errorf("failed to write inline key token %s: %w", k, err)
				}
				if err := enc.WriteValue(v); err != nil {
					return fmt.Errorf("failed to write inline value: %w", err)
				}
			}

		default:
			for k, v := range m {
				if err := enc.WriteToken(jsontext.String(k)); err != nil {
					return fmt.Errorf("failed to write inline key token %s: %w", k, err)
				}
				if err := enc.WriteValue(v); err != nil {
					return fmt.Errorf("failed to write inline value: %w", err)
				}
			}
		}

	case len(w.nestedValue) > 0:
		// Add a new key with the nested value, like "_value": "foo"
		if err := enc.WriteToken(jsontext.String(w.nestedValueKey)); err != nil {
			return fmt.Errorf("failed to write nested value key token %s: %w", w.nestedValueKey, err)
		}
		if err := enc.WriteValue(w.nestedValue); err != nil {
			return fmt.Errorf("failed to write nested value: %w", err)
		}
	default:
		// Don't write anything
	}

	if err := enc.WriteToken(jsontext.ObjectEnd); err != nil {
		return fmt.Errorf("failed to write object end token: %w", err)
	}
	return nil
}

func (w *customKeyWrappedType) UnmarshalJSONV2(dec *jsontext.Decoder, opts json.Options) error {
	if k := dec.PeekKind(); k != '{' {
		return fmt.Errorf("expected object start, but encountered %v", k)
	}

	//  1. Unmarshal to an arbitrary map[string]any
	//  2. Extract the discriminator key+value
	//  3. Use the value from (2) to create a T from the options
	// 4a. If the map contains the nested value key, unmarshal
	//     it into the T from (3)
	// 4b. If the map does not contain the nested value key,
	//     marshal the remainder into JSON, then unmarshal
	//     that JSON into the T from (3)

	m := map[string]any{}
	if err := json.UnmarshalDecode(dec, &m, opts); err != nil {
		return fmt.Errorf("failed to marshal to map[string]any: %w", err)
	}

	dv, ok := m[w.discriminatorKey]
	if !ok {
		return fmt.Errorf(`missing discriminator "%s"`, w.discriminatorKey)
	}
	dvs, ok := dv.(string)
	if !ok {
		return fmt.Errorf(`value for discriminator key "%s" must be a string (got %T)`, w.discriminatorKey, dv)
	}

	// Remove the discriminator from the map
	delete(m, w.discriminatorKey)

	if len(m) == 0 {
		// The map may be empty after unmarshaling if the JSON
		// value was an empty object or a JSON null.
		//
		// In either case, do nothing
		return nil
	}

	var mv any
	isNested := false

	nv, ok := m[w.nestedValueKey]
	if ok {
		isNested = true
		mv = nv
		delete(m, w.nestedValueKey)

		// At this point, we've removed the discriminator and
		// nested values from our map, and it should not have
		// any other values.
		//
		// If it does, that's an error.
		if len(m) > 0 {
			return fmt.Errorf("found both inline and nested values")
		}
	} else {
		mv = m // use the remaining map as the value
	}

	// Re-marshal the value
	b, err := json.Marshal(mv, opts)
	if err != nil {
		return fmt.Errorf("failed to re-marshal value: %w", err)
	}

	// And re-unmarshal it into a jsontext.Value
	var v jsontext.Value
	if err := json.Unmarshal(b, &v, opts); err != nil {
		return fmt.Errorf("failed to unmarshal to jsontext.Value: %w", err)
	}

	if isNested {
		w.nestedValue = v
	} else {
		w.inlineValue = v
	}
	w.discriminatorValue = dvs

	return nil
}
