package go11y

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/caarlos0/env/v10"
)

// Configurator is an interface that defines the methods required for configuration of go11y.
// It is used to abstract the configuration details from the observer implementation.
// This allows for different implementations of configuration, such as loading from environment variables or using a
// custom configuration struct - ideal for our unit tests or when you want to use a your own bespoke configuration
// source
type Configurator interface {
	LogLevel() slog.Level
	OtelURL() string
	ServiceName() string
	TrimPaths() []string
	TrimModules() []string
}

// Configuration is a struct that holds the reference configuration for go11y.
type Configuration struct {
	logLevel    slog.Level
	otelURL     string
	strLevel    string
	databaseURL string
	serviceName string
	trimModules []string
	trimPaths   []string
}

type interimConfig struct {
	StrLevel    string `env:"LOG_LEVEL" envDefault:"debug"`
	OtelURL     string `env:"OTEL_URL" envDefault:""`
	DatabaseURL string `env:"DATABASE_URL" envDefault:""`
	ServiceName string `env:"OTEL_SERVICE_NAME" envDefault:""`
	TrimModules string `env:"TRIM_MODULES" envDefault:""`
	TrimPaths   string `env:"TRIM_PATHS" envDefault:""`
}

// LoadConfig loads the configuration from environment variables.
// It returns a Configuration instance that implements the Configurator interface.
// If any required environment variable is missing or invalid, it returns an error.
func LoadConfig() (cfg *Configuration, fault error) {
	h := interimConfig{}
	if err := env.Parse(&h); err != nil {
		return nil, fmt.Errorf("could not load config: %w", err)
	}

	trimModules := strings.Split(h.TrimModules, ",")

	path, _ := os.Getwd()
	trimPaths := []string{
		path,
	}
	if h.TrimPaths != "" {
		trimPaths = strings.Split(h.TrimPaths, ",")
	}

	c := &Configuration{
		otelURL:     h.OtelURL,
		strLevel:    h.StrLevel,
		logLevel:    StringToLevel(h.StrLevel),
		serviceName: h.ServiceName,
		trimModules: trimModules,
		trimPaths:   trimPaths,
	}

	return c, nil
}

// CreateConfig creates a new Configuration instance populated with the provided parameters.
// This is intended to be used for when you want to create a config without loading from environment variables.
// The Configuration returned satisfies the Configurator interface, allowing it to be used interchangeably with
// configurations loaded from environment variables.
func CreateConfig(logLevel slog.Level, otelURL, dbConStr, serviceName string, trimModules, trimPaths []string) *Configuration {
	return &Configuration{
		logLevel:    logLevel,
		otelURL:     otelURL,
		strLevel:    logLevel.String(),
		databaseURL: dbConStr,
		serviceName: serviceName,
		trimModules: trimModules,
		trimPaths:   trimPaths,
	}
}

// LogLevel returns the configured log level for the observer.
// This method is part of the Configurator interface.
func (c *Configuration) LogLevel() slog.Level {
	return c.logLevel
}

// OtelURL returns the configured OpenTelemetry URL (scheme, host, port, path).
// This method is part of the Configurator interface.
func (c *Configuration) OtelURL() string {
	return c.otelURL
}

// ServiceName returns the configured service name for OpenTelemetry.
// This method is part of the Configurator interface.
func (c *Configuration) ServiceName() string {
	return c.serviceName
}

// TrimPaths returns the configured strings to be trimmed from the source.file attribute.
// This method is part of the Configurator interface.
func (c *Configuration) TrimPaths() []string {
	return c.trimPaths
}

// TrimModules returns the configured strings to be trimmed from the source.function attribute.
// This method is part of the Configurator interface.
func (c *Configuration) TrimModules() []string {
	return c.trimModules
}
