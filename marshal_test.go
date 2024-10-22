package oneof_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/dhoelle/oneof"
	"github.com/go-json-experiment/json"
)

func Test_UnmarshalEmptyObject(t *testing.T) {
	opts := map[string]fmt.Stringer{
		"url.URL": &url.URL{},
	}
	unmarshalFunc := oneof.UnmarshalFunc(opts, nil)

	in := []byte(`{"_type":"url.URL"}`) // no additional fields
	var got fmt.Stringer
	if err := json.Unmarshal(in, &got, json.WithUnmarshalers(unmarshalFunc)); err != nil {
		t.Fatalf("error unmarshaling: %v", err)
	}

	if got == nil {
		t.Fatalf("got is nil")
	}
	u, ok := got.(*url.URL)
	if !ok {
		t.Fatalf("got is not a *url.URL")
	}
	want := url.URL{}
	if *u != want {
		t.Errorf("got != want")
	}
}
