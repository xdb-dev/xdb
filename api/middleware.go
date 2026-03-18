package api

import "net/http"

// Middleware wraps an [http.Handler] with additional behavior.
type Middleware func(next http.Handler) http.Handler

// MiddlewareStack is an ordered list of [Middleware].
// Middleware is applied in reverse order: the last added middleware
// is the outermost wrapper.
type MiddlewareStack []Middleware

// Wrap applies the middleware stack to the given handler.
func (s MiddlewareStack) Wrap(h http.Handler) http.Handler {
	for i := len(s) - 1; i >= 0; i-- {
		h = s[i](h)
	}

	return h
}
