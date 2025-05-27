package config

import (
	"go.uber.org/fx"
)

var KafkaConfigModule = fx.Options(
	fx.Provide(
		NewConfig,
	),
)
