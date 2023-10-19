package crawly

import "errors"

//////////////////////////////////////////////////

var (
	InvalidHandle = errors.New("invalid handle")
)

type Handle interface {
	Equal(handle Handle) bool
	Valid() bool

	String() string
}
