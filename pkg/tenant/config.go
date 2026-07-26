package tenant

// Config holds the configuration for tenant management.
type Config struct {
	Enabled bool `koanf:"enabled"`
}

// ApplyDefaults sets default values for unset configuration fields.
func (c *Config) ApplyDefaults() {

}

// Validate checks if the Config has all required fields set.
func (c *Config) Validate() error {
	return nil
}
