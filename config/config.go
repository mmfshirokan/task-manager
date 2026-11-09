package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort        int           `env:"HTTP_PORT" envDefault:"8080"`
	DatabaseURL     string        `env:"DATABASE_URL,required,notEmpty"`
	JWTSecret       string        `env:"JWT_SECRET,required,notEmpty"`
	JWTTTL          time.Duration `env:"JWT_TTL" envDefault:"24h"`
	OverdueInterval time.Duration `env:"OVERDUE_INTERVAL" envDefault:"1m"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	LogLevel        slog.Level    `env:"LOG_LEVEL" envDefault:"info"`
}

// Load reads variables from envFile (if it exists) and then from the process
// environment. Variables already set in the environment take precedence.
func Load(envFile string) (Config, error) {
	if err := godotenv.Load(envFile); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Config{}, fmt.Errorf("load env file %s: %w", envFile, err)
	}

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("parse env: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validate() error {
	switch {
	case c.HTTPPort <= 0 || c.HTTPPort > 65535:
		return fmt.Errorf("HTTP_PORT must be in range 1..65535, got %d", c.HTTPPort)
	case c.JWTTTL <= 0:
		return fmt.Errorf("JWT_TTL must be positive, got %s", c.JWTTTL)
	case c.OverdueInterval <= 0:
		return fmt.Errorf("OVERDUE_INTERVAL must be positive, got %s", c.OverdueInterval)
	case c.ShutdownTimeout <= 0:
		return fmt.Errorf("SHUTDOWN_TIMEOUT must be positive, got %s", c.ShutdownTimeout)
	}

	return nil
}
