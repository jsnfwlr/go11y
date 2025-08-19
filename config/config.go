package config

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/caarlos0/env/v10"
)

// Config is a struct that holds the reference configuration for go11y.
type Config struct {
	logLevel    slog.Level
	otelURL     string
	strLevel    string
	dbConStr    string
	serviceName string
	trimModules []string
	trimPaths   []string
}

// New creates a new Config instance populated with the provided parameters.
// This is intended to be used for when you want to create a config without loading from environment variables.
// The Config returned satisfies the Configuration interface, allowing it to be used interchangeably with configurations
// loaded from environment variables.
func New(logLevel slog.Level, otelURL, dbConStr, serviceName string, trimModules, trimPaths []string) *Config {
	return &Config{
		logLevel:    logLevel,
		otelURL:     otelURL,
		strLevel:    logLevel.String(),
		dbConStr:    dbConStr,
		serviceName: serviceName,
		trimModules: trimModules,
		trimPaths:   trimPaths,
	}
}

type interimConfig struct {
	StrLevel    string `env:"LOG_LEVEL" envDefault:"debug"`
	OtelURL     string `env:"OTEL_URL" envDefault:""`
	DBConStr    string `env:"DB_CONSTR" envDefault:""`
	ServiceName string `env:"OTEL_SERVICE_NAME" envDefault:""`
	TrimModules string `env:"TRIM_MODULES" envDefault:""`
	TrimPaths   string `env:"TRIM_PATHS" envDefault:""`
}

// Load loads the configuration from environment variables.
// It returns a Config instance that implements the Configuration interface.
// If any required environment variable is missing or invalid, it returns an error.
func Load() (cfg *Config, fault error) {
	h := interimConfig{}
	if err := env.Parse(&h); err != nil {
		return nil, fmt.Errorf("could not load config: %w", err)
	}

	trimModules := strings.Split(h.TrimModules, ",")
	trimPaths := strings.Split(h.TrimPaths, ",")

	c := &Config{
		otelURL:     h.OtelURL,
		dbConStr:    h.DBConStr,
		strLevel:    h.StrLevel,
		logLevel:    StringToLevel(h.StrLevel),
		serviceName: h.ServiceName,
		trimModules: trimModules,
		trimPaths:   trimPaths,
	}

	return c, nil
}

// LogLevel returns the configured log level for the observer.
// This method is part of the Configuration interface.
func (c *Config) LogLevel() slog.Level {
	return c.logLevel
}

// URL returns the configured OpenTelemetry URL (scheme, host, port, path).
// This method is part of the Configuration interface.
func (c *Config) URL() string {
	return c.otelURL
}

// DBConStr returns the database connection string.
// This method is part of the Configuration interface.
func (c *Config) DBConStr() string {
	return c.dbConStr
}

// ServiceName returns the configured service name for OpenTelemetry.
// This method is part of the Configuration interface.
func (c *Config) ServiceName() string {
	return c.serviceName
}

// TrimPaths returns the configured strings to be trimmed from the source.file attribute.
// This method is part of the Configuration interface.
func (c *Config) TrimPaths() []string {
	return c.trimPaths
}

// TrimModules returns the configured strings to be trimmed from the source.function attribute.
// This method is part of the Configuration interface.
func (c *Config) TrimModules() []string {
	return c.trimModules
}

// Configuration is an interface that defines the methods required for configuration of go11y.
// It is used to abstract the configuration details from the observer implementation.
// This allows for different implementations of configuration, such as loading from environment variables or using a
// custom configuration struct - ideal for our unit tests or when you want to use a your own bespoke configuration
// source
type Configuration interface {
	LogLevel() slog.Level
	URL() string
	DBConStr() string
	ServiceName() string
	TrimPaths() []string
	TrimModules() []string
}
