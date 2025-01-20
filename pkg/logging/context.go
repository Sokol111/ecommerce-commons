package handler

type LoggingContext interface {
	RequestMethod() string
	RequestPath() string
	Errors() []error
	Next()
}
