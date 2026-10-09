package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Env      string `env:"APP_ENV" envDefault:"dev"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
	Port     string `env:"PORT" envDefault:"8080"`

	DB  DBConfig
	S3  S3Config  `envPrefix:"S3_"`
	JWT JWTConfig `envPrefix:"JWT_"`

	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" envDefault:"*"`
	MaxUploadSize      int64    `env:"MAX_UPLOAD_SIZE" envDefault:"104857600"`
}

type DBConfig struct {
	URL string `env:"DATABASE_URL,required"`
}

type S3Config struct {
	Endpoint     string `env:"ENDPOINT" envDefault:"http://localhost:9000"`
	Region       string `env:"REGION" envDefault:"us-east-1"`
	AccessKey    string `env:"ACCESS_KEY,required"`
	SecretKey    string `env:"SECRET_KEY,required"`
	Bucket       string `env:"BUCKET" envDefault:"mys3"`
	UsePathStyle bool   `env:"USE_PATH_STYLE" envDefault:"true"`
	Mode         string `env:"MODE" envDefault:"readonly"`
}

type JWTConfig struct {
	Secret     string        `env:"SECRET,required"`
	AccessTTL  time.Duration `env:"ACCESS_TTL" envDefault:"15m"`
	RefreshTTL time.Duration `env:"REFRESH_TTL" envDefault:"720h"`
}

func Load() (*Config, error) {
	loadDotEnv()
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadDotEnv() {
	if err := godotenv.Load(); err != nil {
		return
	}
}

func (c *Config) validate() error {
	switch c.S3.Mode {
	case "readonly", "full":
	default:
		return fmt.Errorf("invalid S3_MODE %q (want readonly|full)", c.S3.Mode)
	}
	return nil
}
