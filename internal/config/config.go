package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv         string
	AppAddr        string
	AppPublicURL   string
	DatabaseURL    string
	JWTSecret      string
	JWTAccessTTL   string
	UploadDir      string
	UploadMaxBytes int64
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not found or already loaded into environment")
	}

	appEnv := getEnv("APP_ENV", "development")
	appAddr := getEnv("APP_ADDR", "127.0.0.1:8080")
	appPublicURL := getEnv("APP_PUBLIC_URL", "http://localhost:3000")
	dbURL := getEnv("DATABASE_URL", "postgres://postgre:admin12345@127.0.0.1:5432/dental_lab?sslmode=disable")
	jwtSecret := getEnv("AUTH_JWT_SECRET", "super_secret_dental_lab_jwt_key_2026")
	jwtAccessTTL := getEnv("AUTH_ACCESS_TTL", "24h")
	uploadDir := getEnv("UPLOAD_DIR", "./uploads")

	return &Config{
		AppEnv:         appEnv,
		AppAddr:        appAddr,
		AppPublicURL:   appPublicURL,
		DatabaseURL:    dbURL,
		JWTSecret:      jwtSecret,
		JWTAccessTTL:   jwtAccessTTL,
		UploadDir:      uploadDir,
		UploadMaxBytes: 15 * 1024 * 1024, // 15MB
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
