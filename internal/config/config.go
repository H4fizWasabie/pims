package config

import (
	"os"
	"strings"
)

type Config struct {
	Port             string
	DatabaseURL      string
	ProcuraDBPath    string
	AdminEmail       string
	AdminPassword    string
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
		ProcuraDBPath:    env("PROCURA_DB_PATH", "/home/procura/data/procura.sqlite"),
		AdminEmail:       os.Getenv("ADMIN_EMAIL"),
		AdminPassword:    os.Getenv("ADMIN_PASSWORD"),
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
	var out []string
	for _, p := range strings.Split(os.Getenv(key), ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
