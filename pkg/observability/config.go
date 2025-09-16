package observability

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	TracingEnabled        bool   `mapstructure:"tracing-enabled"`
	OtelCollectorEndpoint string `mapstructure:"otel-collector-endpoint"`
}

func newConfig(v *viper.Viper) (Config, error) {
	sub := v.Sub("observability")
	if sub == nil {
		return Config{}, fmt.Errorf("observability section missing in config")
	}
	sub.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	sub.SetEnvPrefix("OBSERVABILITY")
	sub.AutomaticEnv()

	var cfg Config
	if err := sub.UnmarshalExact(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load otel config: %w", err)
	}
	return cfg, nil
}
