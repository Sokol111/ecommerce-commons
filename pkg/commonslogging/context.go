package commonslogging

type LoggingContext interface {
	RequestMethod() string
	RequestPath() string
	Errors() []error
	Next()
}
