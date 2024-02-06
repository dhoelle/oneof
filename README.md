# oneof

[![Go Reference](https://pkg.go.dev/badge/github.com/dhoelle/oneof.svg)](https://pkg.go.dev/github.com/dhoelle/oneof)
[![Build Status](https://github.com/dhoelle/oneof/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/dhoelle/oneof/actions)

Package `oneof` enables marshaling and unmarshaling of Go interface values using the Go JSON V2 experiment ([github.com/go-json-experiment/json](https://github.com/go-json-experiment/json)).

By default, marshaling and unmarshaling an interface value will fail:

```go
var s1 fmt.Stringer = crypto.SHA256
b, _ := json.Marshal(s1) // b == []byte("5")

var s2 fmt.Stringer
err := json.Unmarshal(b, &s2)
fmt.Println(err)
// Output:
// json: cannot unmarshal JSON string into Go value of type fmt.Stringer: cannot derive concrete type for non-empty interface
```

`oneof`'s `MarshalFunc` and `UnmarshalFunc` encode matching Go values alongside a type discriminator, which enables round-trip marshaling and unmarshaling:

```go
// Implementations of fmt.Stringer that our program will marshal and unmarshal,
// keyed by an option type
opts := map[string]fmt.Stringer{
  "crypto.Hash": crypto.Hash(0),
  "net.IP":      net.IP{},
  "url.URL":     &url.URL{},
}

marshalFunc := oneof.MarshalFunc(opts, nil)
var s1 fmt.Stringer = crypto.SHA256
b, _ := json.Marshal(s1, json.WithMarshalers(marshalFunc))
// b == []byte(`{"_type": "crypto.Hash", "_value": 5}`)

unmarshalFunc := oneof.UnmarshalFunc(opts, nil)
var s2 fmt.Stringer
_ = json.Unmarshal(b, &s2, json.WithUnmarshalers(unmarshalFunc))
fmt.Printf("unmarshaled type = %T\n", s2)
fmt.Printf("string output = %s\n", s2.String())
// Output:
// unmarshaled type: crypto.Hash
// string output: SHA-256
```

## Default encoding

By default, `MarshalFunc` encodes known Go values into a JSON object where:

- The value of the key `"_type"` is the type discriminator, and,
- The value of the key `"_value"` is the default JSON-encoding of the Go value

For example, given options:

```go
opts := map[string]fmt.Stringer{
  "crypto.Hash": crypto.Hash(0),
  "net.IP":      net.IP{},
  "url.URL":     &url.URL{},
}
```

... the Go value `crypto.SHA256` (which encodes to the JSON number `5`), would be encoded by `MarshalFunc` as:

```json
{
  "_type": "crypto.Hash",
  "_value": 5
}
```

... and the Go value `&url.URL{Scheme: "https", Host: "example.com"}`, which encodes to a JSON object, would be encoded by `MarshalFunc` as:

```jsonc
{
  "_type": "url.URL",
  "_value": {
    "Scheme": "https",
    "Host": "example.com"
    // other url.URL fields omitted
  }
}
```

## Custom encoding

`WrappedValue` is the interface implemented by containers which can marshal and unmarshal Go types including type information. Configure the `WrappedValue` used by `MarshalFunc` and `UnmarshalFunc` by setting `Config.WrapFunc`:

```go
cfg := oneof.Config{
  WrapFunc: oneof.WrapAlwaysNest,
}
marshalFunc := oneof.MarshalFunc(opts, cfg)
```

If `Config.WrapFunc` is unset, `MarshalFunc` and `UnmarshalFunc` default to `WrapNested`, which wraps encoded values under the `"_value"` key.

The `oneof` package also defines `WrapInlineObjects`, which inlines the fields of JSON object values, e.g.,:

```jsonc
{
  "_type": "url.URL",
  "Scheme": "https",
  "Host": "example.com"
  // other url.URL fields omitted
}
```

For finer-grained control, you can create your own `WrappedValue` type, or use `CustomValueWrapper` (whose method `Wrap` can be used as `Config.WrapFunc`)

See the [WrapFunc](https://pkg.go.dev/github.com/dhoelle/oneof/#example_Config_wrapFunc) and [CustomValueWrapper](https://pkg.go.dev/github.com/dhoelle/oneof/#example_CustomValueWrapper) examples.

## Handling missing keys

If `oneof` encounters a Go type for which there is no matching option key while marshaling, it will return an error.

You can override this behavior by setting `Config.ReplaceMissingTypeFunc`:

```go
cfg := &oneof.Config{
  ReplaceMissingTypeFunc: func(v any) string {
    return fmt.Sprintf("MISSING_%T", v)
  },
}
```

With the above `Config`, `MarshalFunc` will use the string produced by `ReplaceMissingTypeFunc` as the option type for missing values, like:

```json
{
  "_type": "MISSING_*crypto.Hash",
  "_value": 5
},
```

> [!NOTE]
> UnmarshalFunc will likely fail to unmarshal output produced by `ReplaceMissingTypeFunc`. If you need to marshal and unmarshal a Go type, include it in the option set.
