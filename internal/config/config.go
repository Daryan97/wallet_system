package config

import (
	"os"      // For environment variables
	"strconv" // For string to int conversion

	"github.com/joho/godotenv" // For loading .env files
)

// Config holds the application configuration
type Config struct {
	AppPort    string // Application port
	DBUser     string // Database user
	DBPassword string // Database password
	DBHost     string // Database host
	DBPort     string // Database port
	DBName     string // Database name
	JWTSecret  string // JWT secret key
	RedisAddr  string // Redis server address
	RedisPass  string // Redis password
	RedisDB    int    // Redis database number
	IsProd     bool   // Is production environment
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	_ = godotenv.Load() // Load .env file if present
	redisDB, _ := strconv.Atoi(os.Getenv("REDIS_DB"))
	return &Config{
		AppPort:    os.Getenv("APP_PORT"),          // Application port
		DBUser:     os.Getenv("DB_USER"),           // Database user
		DBPassword: os.Getenv("DB_PASSWORD"),       // Database password
		DBHost:     os.Getenv("DB_HOST"),           // Database host
		DBPort:     os.Getenv("DB_PORT"),           // Database port
		DBName:     os.Getenv("DB_NAME"),           // Database name
		JWTSecret:  os.Getenv("JWT_SECRET"),        // JWT secret key
		RedisAddr:  os.Getenv("REDIS_ADDR"),        // Redis server address
		RedisPass:  os.Getenv("REDIS_PASS"),        // Redis password
		RedisDB:    redisDB,                        // Redis database number
		IsProd:     os.Getenv("IS_PROD") == "true", // Is production environment
	}
}
