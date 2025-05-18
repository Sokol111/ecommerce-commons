package commonsgin

import (
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

var GinModule = fx.Options(
	fx.Provide(
		ProvideNewEngine,
	),
)

func ProvideNewEngine(log *zap.Logger) http.Handler {
	return NewEngine(log)
}
