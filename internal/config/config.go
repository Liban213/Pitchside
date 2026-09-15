// Package config loads Pitchside's runtime configuration from environment
// variables (see .env.example).
package config

import "os"

type Config struct {
	DatabaseURL       string
	RedisAddr         string
	FootballDataAPIKey string
	Port              string
}

func Load() Config {
	return Config{
		DatabaseURL:        getenv("DATABASE_URL", "postgres://libanm@localhost:5432/pitchside_db"),
		RedisAddr:          getenv("REDIS_ADDR", "localhost:6379"),
		FootballDataAPIKey: os.Getenv("FOOTBALL_DATA_API_KEY"),
		Port:               getenv("PORT", "8080"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
