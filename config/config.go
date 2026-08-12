package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	configDir  = "/config"
	stagingDir = "/downloads"
)

type Config struct {
	Port         string
	DBPath       string
	TMDBLanguage string
	CoverDir     string
	StagingDir   string
	LogLevel     string
	ConfigDir    string

	AccessTTL           time.Duration
	RefreshTTL          time.Duration
	LoginEnabled        bool
	RegistrationEnabled bool

	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRoleClaim    string
	OIDCAdminRole    string
}

func Load() Config {
	_ = godotenv.Load()

	accessTTL := getDuration("MEDIATOR_AUTH_ACCESS_TTL", 15*time.Minute)
	refreshTTL := getDuration("MEDIATOR_AUTH_REFRESH_TTL", 7*24*time.Hour)

	return Config{
		Port:         getEnv("MEDIATOR_PORT", ":42800"),
		DBPath:       filepath.Join(configDir, "media.db"),
		TMDBLanguage: getEnv("MEDIATOR_TMDB_LANGUAGE", "en-US"),
		CoverDir:     filepath.Join(configDir, "covers"),
		StagingDir:   stagingDir,
		LogLevel:     getEnv("MEDIATOR_LOG_LEVEL", "INFO"),
		ConfigDir:    configDir,

		AccessTTL:           accessTTL,
		RefreshTTL:          refreshTTL,
		LoginEnabled:        !getEnvBool("MEDIATOR_AUTH_DISABLE_LOGIN", false),
		RegistrationEnabled: !getEnvBool("MEDIATOR_AUTH_DISABLE_REGISTRATION", false),

		OIDCIssuer:       os.Getenv("MEDIATOR_AUTH_OIDC_ISSUER"),
		OIDCClientID:     os.Getenv("MEDIATOR_AUTH_OIDC_CLIENT_ID"),
		OIDCClientSecret: os.Getenv("MEDIATOR_AUTH_OIDC_CLIENT_SECRET"),
		OIDCRoleClaim:    getEnv("MEDIATOR_AUTH_OIDC_ROLE_CLAIM", "roles"),
		OIDCAdminRole:    getEnv("MEDIATOR_AUTH_OIDC_ADMIN_ROLE", "admin"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := strings.ToLower(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v == "true" || v == "1" || v == "yes"
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
