package config

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Config struct {
	Port      string
	JWTSecret string

	WorkspaceURL string
	ResultURL    string
	AudioURL     string
	VoiceURL     string
	AuthURL      string

	RateLimitRPS float64
	RateLimit    int
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:         strings.TrimSpace(os.Getenv("GATEWAY_SERVICE_PORT")),
		JWTSecret:    strings.TrimSpace(os.Getenv("AUTH_SERVICE_JWT_SECRET")),
		WorkspaceURL: strings.TrimSpace(os.Getenv("GATEWAY_WORKSPACE_SERVICE_URL")),
		ResultURL:    strings.TrimSpace(os.Getenv("GATEWAY_MEETING_SERVICE_URL")),
		AudioURL:     strings.TrimSpace(os.Getenv("GATEWAY_AUDIO_INGEST_SERVICE_URL")),
		VoiceURL:     strings.TrimSpace(os.Getenv("GATEWAY_VOICE_PROFILE_SERVICE_URL")),
		AuthURL:      strings.TrimSpace(os.Getenv("GATEWAY_AUTH_SERVICE_URL")),
	}

	rpsRaw := strings.TrimSpace(os.Getenv("GATEWAY_RATE_LIMIT_RPS"))
	limitRaw := strings.TrimSpace(os.Getenv("GATEWAY_RATE_LIMIT"))

	required := map[string]string{
		"GATEWAY_SERVICE_PORT":              cfg.Port,
		"AUTH_SERVICE_JWT_SECRET":           cfg.JWTSecret,
		"GATEWAY_WORKSPACE_SERVICE_URL":     cfg.WorkspaceURL,
		"GATEWAY_MEETING_SERVICE_URL":       cfg.ResultURL,
		"GATEWAY_AUDIO_INGEST_SERVICE_URL":  cfg.AudioURL,
		"GATEWAY_VOICE_PROFILE_SERVICE_URL": cfg.VoiceURL,
		"GATEWAY_AUTH_SERVICE_URL":          cfg.AuthURL,
		"GATEWAY_RATE_LIMIT_RPS":            rpsRaw,
		"GATEWAY_RATE_LIMIT":                limitRaw,
	}

	missing := make([]string, 0)
	for k, v := range required {
		if v == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}

	rps, err := strconv.ParseFloat(rpsRaw, 64)
	if err != nil {
		return nil, fmt.Errorf("GATEWAY_RATE_LIMIT_RPS must be float, got %q: %w", rpsRaw, err)
	}
	if rps <= 0 {
		return nil, fmt.Errorf("GATEWAY_RATE_LIMIT_RPS must be > 0, got %v", rps)
	}

	limit, err := strconv.Atoi(limitRaw)
	if err != nil {
		return nil, fmt.Errorf("GATEWAY_RATE_LIMIT must be int, got %q: %w", limitRaw, err)
	}
	if limit <= 0 {
		return nil, fmt.Errorf("GATEWAY_RATE_LIMIT must be > 0, got %d", limit)
	}

	cfg.RateLimitRPS = rps
	cfg.RateLimit = limit

	return cfg, nil
}
