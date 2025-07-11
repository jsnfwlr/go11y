package o11y

import (
	"fmt"
	"log/slog"

	"github.com/caarlos0/env/v10"
)

type Config struct {
	otelHost string
	otelPort string
	strLevel string
	dbConStr string
	logLevel slog.Level
}

func (c Config) LogLevel() slog.Level {
	return c.logLevel
}

func (c Config) OtelHost() string {
	return c.otelHost
}

func (c Config) OtelPort() string {
	return c.otelPort
}

func (c Config) DBConStr() string {
	return c.dbConStr
}

type hiddenConfig struct {
	OtelHost string `env:"OTEL_HOST" envDefault:""`
	OtelPort string `env:"OTEL_PORT" envDefault:""`
	DBConStr string `env:"DB_CONSTR" envDefault:""`
	strLevel string `env:"LOG_LEVEL" envDefault:"debug"`
}

func LoadConfig() (cfg Config, fault error) {
	h := hiddenConfig{}
	if err := env.Parse(&h); err != nil {
		return Config{}, fmt.Errorf("could not load config: %w", err)
	}

	c := Config{
		otelHost: h.OtelHost,
		otelPort: h.OtelPort,
		dbConStr: h.DBConStr,
		strLevel: h.strLevel,
		logLevel: StringToLevel(h.strLevel),
	}

	return c, nil
}

type Configuration interface {
	LogLevel() slog.Level
	OtelHost() string
	OtelPort() string
	DBConStr() string
}
