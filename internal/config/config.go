package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	AppEnv             string
	Port               string
	DatabaseURL        string
	JWTSecret          string
	CORSAllowedOrigins []string

	// Admin seed
	AdminSeedEmail    string
	AdminSeedPassword string
	AdminSeedName     string

	// Supabase Storage (future use)
	SupabaseStorageURL     string
	SupabaseStorageBucket  string
	SupabaseServiceRoleKey string

	// Sync Worker
	SyncWorkerEnabled  bool
	SyncWorkerInterval int
}

// Load reads environment variables and returns a Config struct.
// In development, it loads from .env file first.
func Load() *Config {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	// Load .env file in development only
	if env == "development" {
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found, using environment variables")
		}
	}

	cfg := &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		JWTSecret:   getEnv("JWT_SECRET", ""),

		AdminSeedEmail:    getEnv("ADMIN_SEED_EMAIL", "admin@marketops.local"),
		AdminSeedPassword: getEnv("ADMIN_SEED_PASSWORD", ""),
		AdminSeedName:     getEnv("ADMIN_SEED_NAME", "Admin"),

		SupabaseStorageURL:     getEnv("SUPABASE_STORAGE_URL", ""),
		SupabaseStorageBucket:  getEnv("SUPABASE_STORAGE_BUCKET", "product-images"),
		SupabaseServiceRoleKey: getEnv("SUPABASE_SERVICE_ROLE_KEY", ""),
		SyncWorkerEnabled:      getEnv("SYNC_WORKER_ENABLED", "false") == "true",
		SyncWorkerInterval:     getEnvInt("SYNC_WORKER_INTERVAL_SECONDS", 300),
	}

	// Parse CORS origins
	corsRaw := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")
	cfg.CORSAllowedOrigins = parseCORSOrigins(corsRaw)

	// Validate critical config
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	if cfg.IsProduction() {
		if cfg.JWTSecret == "your-jwt-secret-change-in-production" {
			log.Fatal("SECURITY ERROR: JWT_SECRET must be changed from the default value in production!")
		}
		if cfg.AdminSeedPassword == "Admin123!" {
			log.Fatal("SECURITY ERROR: ADMIN_SEED_PASSWORD must be changed from the default value in production!")
		}
		if len(cfg.JWTSecret) < 32 {
			log.Println("WARNING: JWT_SECRET should ideally be at least 32 characters long in production.")
		}
	}

	return cfg
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func parseCORSOrigins(raw string) []string {
	origins := strings.Split(raw, ",")
	result := make([]string, 0, len(origins))
	for _, o := range origins {
		trimmed := strings.TrimSpace(o)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func getEnvInt(key string, fallback int) int {
	valStr := getEnv(key, "")
	if valStr == "" {
		return fallback
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return fallback
	}
	return val
}
