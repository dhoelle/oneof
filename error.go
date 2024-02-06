package oneof

import "fmt"

// ErrUnknownGoType is the error returned by MarshalFunc when it encounters a Go
// type that is not in the provided set of options
type ErrUnknownGoType struct {
	typ string
}

func (e ErrUnknownGoType) Error() string {
	return fmt.Sprintf("unknown Go type %s", e.typ)
}

// ErrUnknownDiscriminatorValue is the error returned by UnmarshalFunc when it
// encounters a JSON discriminator value which is not in the provided set of
// options
type ErrUnknownDiscriminatorValue struct {
	v string
}

func (e ErrUnknownDiscriminatorValue) Error() string {
	return fmt.Sprintf("unknown discriminator value %s", e.v)
}
