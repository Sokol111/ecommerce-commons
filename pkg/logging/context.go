package logging

type LoggingContext interface {
	RequestMethod() string
	RequestPath() string
	Errors() []error
	Next()
}
