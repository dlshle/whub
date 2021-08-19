package middleware

const MaxMiddlewareCount = 64

var Middlewares []RequestMiddleware

func init() {
	Middlewares = make([]RequestMiddleware, 0, 64)
}
