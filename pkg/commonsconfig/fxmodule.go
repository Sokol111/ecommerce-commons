package commonsconfig

import "go.uber.org/fx"

var ViperModule = fx.Options(
	fx.Provide(NewViper),
)
