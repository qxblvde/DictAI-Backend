package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port             string
	PostgresDSN      string
	MinIOEndpoint    string
	MinIOAccess      string
	MinIOSecret      string
	MinIOBucket      string
	MinIOAudioBucket string
	MinIOUseSSL      bool
}

func Load() (*Config, error) {
	port := getEnv("RESULTS_SERVICE_PORT", "8080")

	pgUser := required("POSTGRES_USER")
	pgPass := required("POSTGRES_PASSWORD")
	pgHost := required("POSTGRES_HOST")
	pgPort := getEnv("POSTGRES_PORT", "5432")
	pgDB := required("POSTGRES_DB")
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", pgUser, pgPass, pgHost, pgPort, pgDB)

	minioSSL, _ := strconv.ParseBool(os.Getenv("MINIO_USE_SSL"))

	return &Config{
		Port:             port,
		PostgresDSN:      dsn,
		MinIOEndpoint:    required("RESULTS_SERVICE_MINIO_ENDPOINT"),
		MinIOAccess:      required("MINIO_ACCESS_KEY"),
		MinIOSecret:      required("MINIO_SECRET_KEY"),
		MinIOBucket:      getEnv("RESULTS_SERVICE_MINIO_BUCKET", "results"),
		MinIOAudioBucket: getEnv("RESULTS_SERVICE_AUDIO_MINIO_BUCKET", "audio"),
		MinIOUseSSL:      minioSSL,
	}, nil
}

func required(name string) string {
	v := os.Getenv(name)
	if v == "" {
		panic("required env var not set: " + name)
	}
	return v
}

func getEnv(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}
