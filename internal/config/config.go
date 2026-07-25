package config

import "os"

type Config struct {
	Port             string
	DatabaseURL      string
	SessionSecret    string
	OpenRouterAPIKey string
	OpenRouterModel  string
	GeminiAPIKey     string
	IndentApprovers  []string
	SpecApprovers    []string
	MasterAdmins     []string
}

func Load() *Config {
	return &Config{
		Port:             env("PORT", "8083"),
		DatabaseURL:      env("DATABASE_URL", "postgres://pims:pims@localhost:5432/pims?sslmode=disable"),
		SessionSecret:    env("SESSION_SECRET", "dev-secret-change-me"),
		OpenRouterAPIKey: os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModel:  env("OPENROUTER_MODEL", "google/gemma-4-31b-it:free"),
		GeminiAPIKey:     os.Getenv("GEMINI_API_KEY"),
		IndentApprovers:  splitEnv("INDENT_APPROVERS"),
		SpecApprovers:    splitEnv("SPEC_APPROVERS"),
		MasterAdmins:     splitEnv("MASTER_ADMINS"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitEnv(key string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}
	parts := []string{}
	for _, p := range splitRaw(raw) {
		if t := trimSpace(p); t != "" {
			parts = append(parts, t)
		}
	}
	return parts
}

func splitRaw(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func trimSpace(s string) string {
	for len(s) > 0 && s[0] == ' ' {
		s = s[1:]
	}
	for len(s) > 0 && s[len(s)-1] == ' ' {
		s = s[:len(s)-1]
	}
	return s
}
