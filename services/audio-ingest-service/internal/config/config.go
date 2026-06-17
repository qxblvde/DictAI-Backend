package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                   string
	GRPCPort               string
	JWTSecret              string
	WorkspaceServiceURL    string
	VoiceProfileServiceURL string
	ResultsServiceURL      string
	PostgresDSN            string
	NATSURL                string

	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOUseSSL    bool
}

func Load() *Config {
	return &Config{
		Port:                   requiredEnv("AUDIO_INGEST_SERVICE_PORT"),
		GRPCPort:               requiredEnv("AUDIO_INGEST_SERVICE_GRPC_PORT"),
		JWTSecret:              requiredEnv("AUTH_SERVICE_JWT_SECRET"),
		WorkspaceServiceURL:    requiredEnv("AUDIO_INGEST_WORKSPACE_SERVICE_URL"),
		VoiceProfileServiceURL: requiredEnv("AUDIO_INGEST_VOICE_PROFILE_SERVICE_URL"),
		ResultsServiceURL:      os.Getenv("RESULTS_SERVICE_URL"),
		PostgresDSN:            requiredEnv("AUDIO_INGEST_POSTGRES_DSN"),
		NATSURL:                requiredEnv("NATS_URL"),

		MinIOEndpoint:  requiredEnv("AUDIO_INGEST_MINIO_ENDPOINT"),
		MinIOAccessKey: requiredEnv("MINIO_ACCESS_KEY"),
		MinIOSecretKey: requiredEnv("MINIO_SECRET_KEY"),
		MinIOBucket:    requiredEnv("AUDIO_INGEST_MINIO_BUCKET"),
		MinIOUseSSL:    requiredBoolEnv("MINIO_USE_SSL"),
	}
}

func requiredEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("%s not set", key)
	}

	return value
}

func requiredBoolEnv(key string) bool {
	raw := requiredEnv(key)

	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		log.Fatalf("%s must be bool, got %q", key, raw)
	}

	return parsed
}
