package errors

import (
	"fmt"
	"net/http"
)

func SeeOther(u string) *redirect {
	return &redirect{s: http.StatusSeeOther, u: u}
}
func NewSeeOther(e error, u string) *redirect {
	return &redirect{c: e, s: http.StatusSeeOther, u: u}
}
func Found(u string) *redirect {
	return &redirect{s: http.StatusFound, u: u}
}
func NewFound(e error, u string) *redirect {
	return &redirect{c: e, s: http.StatusFound, u: u}
}
func MovedPermanently(u string) *redirect {
	return &redirect{s: http.StatusMovedPermanently, u: u}
}
func NewMovedPermanently(e error, u string) *redirect {
	return &redirect{c: e, s: http.StatusMovedPermanently, u: u}
}
func NotModified(u string) *redirect {
	return &redirect{s: http.StatusNotModified, u: u}
}
func NewNotModified(e error, u string) *redirect {
	return &redirect{c: e, s: http.StatusNotModified, u: u}
}
func TemporaryRedirect(u string) *redirect {
	return &redirect{s: http.StatusTemporaryRedirect, u: u}
}
func NewTemporaryRedirect(e error, u string) *redirect {
	return &redirect{c: e, s: http.StatusTemporaryRedirect, u: u}
}
func PermanentRedirect(u string) *redirect {
	return &redirect{s: http.StatusPermanentRedirect, u: u}
}
func NewPermanentRedirect(e error, u string) *redirect {
	return &redirect{c: e, s: http.StatusPermanentRedirect, u: u}
}

type redirect struct {
	c error
	u string
	s int
}

func (r redirect) Error() string {
	if r.c == nil {
		return fmt.Sprintf("Redirect %d to %s", r.s, r.u)
	}
	return fmt.Sprintf("Redirect %d to %s: %s", r.s, r.u, r.c)
}

// As is used by the errors.As() function to coerce the method's parameter to the one of the receiver
//
//	if the underlying logic of the receiver's type can understand it.
//
// In this case we're converting a forbidden to its underlying type Err.
func (r *redirect) As(err interface{}) bool {
	switch x := err.(type) {
	case **redirect:
		*x = r
	case *redirect:
		*x = *r
	default:
		return false
	}
	return true
}

func (r redirect) Is(e error) bool {
	rr := redirect{}
	return As(e, &rr) && r.s == rr.s
}

func IsRedirect(e error) bool {
	_, okp := e.(*redirect)
	_, oks := e.(redirect)
	return okp || oks || As(e, &redirect{})
}

func IsNotModified(e error) bool {
	ep, okp := e.(*redirect)
	es, oks := e.(redirect)

	ae := redirect{}
	return (okp && ep.s == http.StatusNotModified) ||
		(oks && es.s == http.StatusNotModified) ||
		(As(e, &ae) && ae.s == http.StatusNotModified)
}

func (r redirect) Unwrap() error {
	return r.c
}
