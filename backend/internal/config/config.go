package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                string
	Env                 string
	DBDialect           string
	DBName              string
	DBHost              string
	DBPort              string
	DBUser              string
	DBPassword          string
	DBSSLMode           string
	JWTSecret           string
	JWTExpiryHours      int
	CORSAllowedOrigins  string
	ReadTimeoutSeconds  int
	WriteTimeoutSeconds int
	IdleTimeoutSeconds  int
}

func Load() *Config {
	// Load .env file if it exists, otherwise fall back to environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from system environment variables")
	}

	jwtExpiryStr := getEnv("JWT_EXPIRY_HOURS", "8")
	jwtExpiry, err := strconv.Atoi(jwtExpiryStr)
	if err != nil {
		jwtExpiry = 8
	}

	return &Config{
		Port:                getEnv("PORT", "8080"),
		Env:                 getEnv("ENV", "development"),
		DBDialect:           getEnv("DB_DIALECT", "sqlite"),
		DBName:              getEnv("DB_NAME", "police_fir.db"),
		DBHost:              getEnv("DB_HOST", "127.0.0.1"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              getEnv("DB_USER", ""),
		DBPassword:          getEnv("DB_PASSWORD", ""),
		DBSSLMode:           getEnv("DB_SSLMODE", "disable"),
		JWTSecret:           getEnv("JWT_SECRET", "supersecretkey_change_me"),
		JWTExpiryHours:      jwtExpiry,
		CORSAllowedOrigins:  getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000"),
		ReadTimeoutSeconds:  getEnvInt("READ_TIMEOUT_SECONDS", 10),
		WriteTimeoutSeconds: getEnvInt("WRITE_TIMEOUT_SECONDS", 30),
		IdleTimeoutSeconds:  getEnvInt("IDLE_TIMEOUT_SECONDS", 60),
	}
}

func (c *Config) Validate() error {
	if c.JWTExpiryHours <= 0 || c.JWTExpiryHours > 12 {
		return fmt.Errorf("JWT_EXPIRY_HOURS must be between 1 and 12")
	}
	if strings.EqualFold(c.Env, "production") {
		if c.JWTSecret == "" || c.JWTSecret == "supersecretkey_change_me" {
			return fmt.Errorf("JWT_SECRET must be set to a strong non-default value")
		}
		if len(c.JWTSecret) < 32 {
			return fmt.Errorf("JWT_SECRET must be at least 32 characters")
		}
		if strings.Contains(c.CORSAllowedOrigins, "*") {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS cannot contain * in production")
		}
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	raw := getEnv(key, strconv.Itoa(fallback))
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
