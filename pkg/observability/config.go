package observability

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	ServiceName    string `mapstructure:"service-name"`
	ServiceVersion string `mapstructure:"service-version"`
	Endpoint       string `mapstructure:"endpoint"`
}

func newConfig(v *viper.Viper) (Config, error) {
	var cfg Config
	if err := v.Sub("otel").UnmarshalExact(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to load otel config: %w", err)
	}
	return cfg, nil
}
