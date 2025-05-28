package swaggerui

import "go.uber.org/fx"

var SwaggerModule = fx.Options(
	fx.Invoke(registerSwaggerUI),
)
