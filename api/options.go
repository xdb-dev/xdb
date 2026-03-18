package api

// EndpointOption configures an [Endpoint].
type EndpointOption interface {
	apply(o *options)
}

type optionFunc func(o *options)

func (f optionFunc) apply(o *options) { f(o) }

type options struct {
	errorHandler ErrorHandler
	middleware   MiddlewareStack
}

func defaultOptions() *options {
	return &options{
		errorHandler: &defaultErrorHandler{},
	}
}

// WithErrorHandler sets a custom [ErrorHandler] for the endpoint.
func WithErrorHandler(h ErrorHandler) EndpointOption {
	return optionFunc(func(o *options) {
		o.errorHandler = h
	})
}

// WithMiddleware appends middleware to the endpoint's middleware stack.
func WithMiddleware(mw ...Middleware) EndpointOption {
	return optionFunc(func(o *options) {
		o.middleware = append(o.middleware, mw...)
	})
}
