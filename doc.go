// SPDX-FileCopyrightText: © 2024 Donald Hoelle. All rights reserved.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package [oneof] enables marshaling and unmarshaling of Go interface values
// using the Go JSON V2 experiment ([github.com/go-json-experiment/json]).
//
// By default, marshaling and unmarshaling an interface value will fail:
//
//	var s1 fmt.Stringer = crypto.SHA256
//	b, _ := json.Marshal(s1) // b == []byte("5")
//
//	var s2 fmt.Stringer
//	err := json.Unmarshal(b, &s2)
//	fmt.Println(err)
//	// Output:
//	// json: cannot unmarshal JSON string into Go value of type fmt.Stringer: cannot derive concrete type for non-empty interface
//
// [MarshalFunc] and [UnmarshalFunc] encode matching Go values alongside a type
// discriminator, which enables round-trip marshaling and unmarshaling:
//
//	// Implementations of fmt.Stringer that our program will marshal and unmarshal,
//	// keyed by an option type
//	opts := map[string]fmt.Stringer{
//	  "crypto.Hash": crypto.Hash(0),
//	  "net.IP":      net.IP{},
//	  "url.URL":     &url.URL{},
//	}
//
//	marshalFunc := oneof.MarshalFunc(opts, nil)
//	var s1 fmt.Stringer = crypto.SHA256
//	b, _ := json.Marshal(s1, json.WithMarshalers(marshalFunc))
//	// b == []byte(`{"_type": "crypto.Hash", "_value": 5}`)
//
//	unmarshalFunc := oneof.UnmarshalFunc(opts, nil)
//	var s2 fmt.Stringer
//	_ = json.Unmarshal(b, &s2, json.WithUnmarshalers(unmarshalFunc))
//	fmt.Printf("unmarshaled type = %T\n", s2)
//	fmt.Printf("string output = %s\n", s2.String())
//	// Output:
//	// unmarshaled type: crypto.Hash
//	// string output: SHA-256
//
// # Default encoding
//
// By default, [MarshalFunc] encodes Go values into a JSON object where:
//
//   - The value of the key "_type" is the type discriminator, and,
//   - The value of the key "_value" is the default JSON-encoding of the Go
//     value
//
// For example, given options:
//
//	opts := map[string]fmt.Stringer{
//	  "crypto.Hash": crypto.Hash(0),
//	  "net.IP":      net.IP{},
//	  "url.URL":     &url.URL{},
//	}
//
// ... the Go value crypto.SHA256, which encodes to the JSON number 5, would be
// encoded by [MarshalFunc] as:
//
//	{
//	  "_type": "crypto.Hash",
//	  "_value": 5
//	}
//
// ... and the Go value &url.URL{Scheme: "https", Host: "example.com"}, which
// encodes to a JSON object, would be encoded by [MarshalFunc] as:
//
//	{
//	  "_type": "url.URL",
//	  "_value": {
//	    "Scheme": "https",
//	    "Host": "example.com"
//	    // other url.URL fields omitted
//	  }
//	}
//
// # Custom encoding
//
// [WrappedValue] is the interface implemented by containers which can marshal
// and unmarshal Go types including type information. Configure the
// [WrappedValue] used by [MarshalFunc] and [UnmarshalFunc] by providing a
// non-nil WrapFunc to [Config]:
//
//	cfg := oneof.Config{
//	  WrapFunc: oneof.WrapAlwaysNest,
//	}
//	marshalFunc := oneof.MarshalFunc(opts, cfg)
//
// If WrapFunc is nil, [MarshalFunc] and [UnmarshalFunc] default to
// [WrapNested], which wraps encoded values under the "_value" key.
//
// [oneof] also defines [WrapInlineObjects], which inlines the fields of JSON
// object values, e.g.,:
//
//	{
//	  "_type": "url.URL",
//	  "Scheme": "https",
//	  "Host": "example.com"
//	  // other url.URL fields omitted
//	}
//
// For finer-grained control, you can create your own implementation of
// [WrappedValue], or use [CustomValueWrapper] ([CustomValueWrapper.Wrap] can be
// used as the WrapFunc in [Config])
//
// See the [Config] and [CustomValueWrapper] examples for more details.
//
// # Handling missing keys
//
// If [oneof] encounters a Go type for which there is no matching option key
// while marshaling, it will return an error.
//
// You can override this behavior by setting the ReplaceMissingTypeFunc field of
// [Config]:
//
//	cfg := &oneof.Config{
//	  ReplaceMissingTypeFunc: func(v any) string {
//	    return fmt.Sprintf("MISSING_%T", v)
//	  },
//	}
//
// With the above [Config], [MarshalFunc] will use the string produced by
// ReplaceMissingTypeFunc as the option type for missing values, like:
//
//	{
//	  "_type": "MISSING_*crypto.Hash",
//	  "_value": 5
//	},
//
// Note that [UnmarshalFunc] will likely fail to unmarshal output produced by
// ReplaceMissingTypeFunc. If you need to marshal and unmarshal a Go type,
// include it in the option set.
//
// [github.com/go-json-experiment/json]: https://github.com/go-json-experiment/json
package oneof
