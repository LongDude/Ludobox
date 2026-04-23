package config

import (
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	PostgresConfig             PostgresConfig
	RedisConfig                RedisConfig
	HttpServerConfig           HTTPServerConfig
	Domain                     string   `env:"DOMAIN" env-default:"localhost"`
	PublicURL                  string   `env:"PUBLIC_URL"`
	SSOAuthenticateURL         string   `env:"SSO_AUTHENTICATE_URL" env-default:"http://sso-core:8080/api/auth/authenticate"`
	MatchmakingSelectServerURL string   `env:"MATCHMAKING_SELECT_SERVER_URL" env-default:"http://haproxy:80/__internal/matchmaking/game-servers/next"`
	InternalProxyToken         string   `env:"INTERNAL_PROXY_TOKEN" env-default:"rooms-internal-token"`
	SwaggerBasePath            string   `env:"SWAGGER_BASE_PATH" env-default:"/api"`
	AllowedCORSOrigins         []string `env:"ALLOWED_CORS_ORIGINS" env-separator:","`
	AllowedRedirectURLs        []string `env:"ALLOWED_REDIRECT_URLS" env-separator:","`
	DefaultAdminEmails         []string `env:"DEFAULT_ADMIN_EMAILS" env-separator:","`
	LogLevel                   string   `env:"LOG_LEVEL" env-default:"info"`
	SwaggerEnabled             bool     `env:"SWAGGER_ENABLED" env-default:"true"`
	SwaggerUser                string   `env:"SWAGGER_USER"`
	SwaggerPassword            string   `env:"SWAGGER_PASSWORD"`
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
