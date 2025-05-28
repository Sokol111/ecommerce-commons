package logging

import (
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

var ZapLoggingModule = fx.Options(
	fx.Provide(zap.NewExample),
	fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
		return &fxevent.ZapLogger{Logger: log}
	}),
)
