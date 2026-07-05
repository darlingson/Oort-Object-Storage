package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	AppPort string

	JWTSecret     string
	AdminEmail    string
	AdminPassword string
}

var globalConfig *Config

func Get() *Config {
	return globalConfig
}

func Load() *Config {

	err := godotenv.Load()

	if err != nil {
		log.Println(".env file not found")
	}

	globalConfig = &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "masterpassword"),
		DBName:     getEnv("DB_NAME", "oort_objects"),

		AppPort: getEnv("APP_PORT", "3333"),

		JWTSecret:     getEnv("JWT_SECRET", "change-me"),
		AdminEmail:    getEnv("ADMIN_EMAIL", "admin@oort.local"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
	}

	return globalConfig
}

func getEnv(key string, fallback string) string {

	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}