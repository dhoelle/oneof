package oneof_test

import (
	"crypto"
	"fmt"
	"net"
	"net/url"

	"github.com/dhoelle/oneof"
	"github.com/go-json-experiment/json"
)

func Example_simpleRoundTrip() {
	// Implementations of fmt.Stringer that our program will marshal and
	// unmarshal, keyed by an option type
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
	// unmarshaled type = crypto.Hash
	// string output = SHA-256
}
