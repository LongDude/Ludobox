package config

import (
	"user_service/internal/types"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	PostgresConfig         PostgresConfig
	RedisConfig            RedisConfig
	HttpServerConfig       HTTPServerConfig
	GameServerStaleAfter   types.CustomDuration `env:"GAME_SERVER_STALE_AFTER" env-default:"15s"`
	RecommendationCacheTTL types.CustomDuration `env:"MATCHMAKING_RECOMMENDATION_CACHE_TTL" env-default:"10s"`
	Domain                 string               `env:"DOMAIN" env-default:"localhost"`
	PublicURL              string               `env:"PUBLIC_URL"`
	SwaggerBasePath        string               `env:"SWAGGER_BASE_PATH" env-default:"/api"`
	AllowedCORSOrigins     []string             `env:"ALLOWED_CORS_ORIGINS" env-separator:","`
	AllowedRedirectURLs    []string             `env:"ALLOWED_REDIRECT_URLS" env-separator:","`
	DefaultAdminEmails     []string             `env:"DEFAULT_ADMIN_EMAILS" env-separator:","`
	SwaggerEnabled         bool                 `env:"SWAGGER_ENABLED" env-default:"true"`
	SwaggerUser            string               `env:"SWAGGER_USER"`
	SwaggerPassword        string               `env:"SWAGGER_PASSWORD"`
}

type PostgresConfig struct {
	Host     string `env:"POSTGRES_HOST" env-required:"true"`
	Port     int    `env:"POSTGRES_PORT" env-required:"true"`
	User     string `env:"POSTGRES_USER" env-required:"true"`
	Password string `env:"POSTGRES_PASSWORD" env-required:"true"`
	DBName   string `env:"POSTGRES_DB" env-required:"true"`
	SSLMode  string `env:"POSTGRES_SSL" env-default:"disable"`
}

type RedisConfig struct {
	Host     string `env:"REDIS_HOST" env-required:"true"`
	Port     string `env:"REDIS_PORT" env-required:"true"`
	Password string `env:"REDIS_PASSWORD" env-required:"true"`
	DB       int    `env:"REDIS_DB" env-default:"0"`
}

type HTTPServerConfig struct {
	Port string `env:"HTTP_PORT" env-default:"8080"`
}

func MustLoadConfig() (*Config, error) {
	var config Config
	if err := cleanenv.ReadEnv(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
