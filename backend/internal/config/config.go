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
	Port                 string
	Env                  string
	DBDialect            string
	DBName               string
	DBHost               string
	DBPort               string
	DBUser               string
	DBPassword           string
	DBSSLMode            string
	JWTSecret            string
	JWTExpiryHours       int
	CORSAllowedOrigins   string
	ReadTimeoutSeconds   int
	WriteTimeoutSeconds  int
	IdleTimeoutSeconds   int
	AIEnabled            bool
	AIProvider           string
	AIBaseURL            string
	AIModel              string
	AIAPIKey             string
	TranslationBaseURL   string
	EmbeddingBaseURL     string
	SearchBaseURL        string
	ObjectStoreEndpoint  string
	SarvamAPIKey         string
	AuthMode             string
	OIDCIssuer           string
	OIDCAudience         string
	OIDCJWKSURL          string
	OPAURL               string
	SearchIndex          string
	SearchUsername       string
	SearchPassword       string
	ObjectStoreAccessKey string
	ObjectStoreSecretKey string
	ObjectStoreBucket    string
	ObjectStoreRegion    string
	GraphBaseURL         string
	GraphUsername        string
	GraphPassword        string
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
		Port:                 getEnv("PORT", "8002"),
		Env:                  getEnv("ENV", "development"),
		DBDialect:            getEnv("DB_DIALECT", "sqlite"),
		DBName:               getEnv("DB_NAME", "police_fir.db"),
		DBHost:               getEnv("DB_HOST", "127.0.0.1"),
		DBPort:               getEnv("DB_PORT", "5432"),
		DBUser:               getEnv("DB_USER", ""),
		DBPassword:           getEnv("DB_PASSWORD", ""),
		DBSSLMode:            getEnv("DB_SSLMODE", "disable"),
		JWTSecret:            getEnv("JWT_SECRET", "supersecretkey_change_me"),
		JWTExpiryHours:       jwtExpiry,
		CORSAllowedOrigins:   getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173,http://localhost:3000,http://127.0.0.1:3000"),
		ReadTimeoutSeconds:   getEnvInt("READ_TIMEOUT_SECONDS", 10),
		WriteTimeoutSeconds:  getEnvInt("WRITE_TIMEOUT_SECONDS", 30),
		IdleTimeoutSeconds:   getEnvInt("IDLE_TIMEOUT_SECONDS", 60),
		AIEnabled:            getEnvBool("AI_ENABLED", false),
		AIProvider:           getEnv("AI_PROVIDER", "gemini"),
		AIBaseURL:            getEnv("AI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta"),
		AIModel:              getEnv("AI_MODEL", "gemini-3.5-flash"),
		AIAPIKey:             getEnv("AI_API_KEY", ""),
		TranslationBaseURL:   getEnv("TRANSLATION_BASE_URL", "https://api.sarvam.ai"),
		EmbeddingBaseURL:     getEnv("EMBEDDING_BASE_URL", ""),
		SearchBaseURL:        getEnv("SEARCH_BASE_URL", ""),
		ObjectStoreEndpoint:  getEnv("OBJECT_STORAGE_ENDPOINT", ""),
		SarvamAPIKey:         getEnv("SARVAM_API_KEY", ""),
		AuthMode:             getEnv("AUTH_MODE", "local"),
		OIDCIssuer:           getEnv("OIDC_ISSUER", ""),
		OIDCAudience:         getEnv("OIDC_AUDIENCE", "crime-api"),
		OIDCJWKSURL:          getEnv("OIDC_JWKS_URL", ""),
		OPAURL:               getEnv("OPA_URL", ""),
		SearchIndex:          getEnv("SEARCH_INDEX", "police-cases"),
		SearchUsername:       getEnv("SEARCH_USERNAME", ""),
		SearchPassword:       getEnv("SEARCH_PASSWORD", ""),
		ObjectStoreAccessKey: getEnv("OBJECT_STORAGE_ACCESS_KEY", ""),
		ObjectStoreSecretKey: getEnv("OBJECT_STORAGE_SECRET_KEY", ""),
		ObjectStoreBucket:    getEnv("OBJECT_STORAGE_BUCKET", "police-evidence"),
		ObjectStoreRegion:    getEnv("OBJECT_STORAGE_REGION", "us-east-1"),
		GraphBaseURL:         getEnv("GRAPH_BASE_URL", ""),
		GraphUsername:        getEnv("GRAPH_USERNAME", "neo4j"),
		GraphPassword:        getEnv("GRAPH_PASSWORD", ""),
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
	if c.AIEnabled {
		if strings.TrimSpace(c.AIBaseURL) == "" || strings.TrimSpace(c.AIModel) == "" || strings.TrimSpace(c.AIAPIKey) == "" {
			return fmt.Errorf("AI_BASE_URL, AI_MODEL and AI_API_KEY are required when AI_ENABLED=true")
		}
		if strings.TrimSpace(c.TranslationBaseURL) == "" {
			return fmt.Errorf("TRANSLATION_BASE_URL is required when AI_ENABLED=true")
		}
		if strings.TrimSpace(c.SarvamAPIKey) == "" {
			return fmt.Errorf("SARVAM_API_KEY is required when AI_ENABLED=true")
		}
	}
	if c.AuthMode != "local" && c.AuthMode != "oidc" {
		return fmt.Errorf("AUTH_MODE must be local or oidc")
	}
	if c.AuthMode == "oidc" && (strings.TrimSpace(c.OIDCIssuer) == "" || strings.TrimSpace(c.OIDCJWKSURL) == "") {
		return fmt.Errorf("OIDC_ISSUER and OIDC_JWKS_URL are required when AUTH_MODE=oidc")
	}
	return nil
}

func getEnvBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(getEnv(key, strconv.FormatBool(fallback))))
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
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
