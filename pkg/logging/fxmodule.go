package logging

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ZapLoggingModule = fx.Options(
	fx.Provide(zap.NewExample),
)
