package kafka

import (
	"go.uber.org/fx"
)

var KafkaModule = fx.Options(
	fx.Provide(
		NewConfig,
	),
)
