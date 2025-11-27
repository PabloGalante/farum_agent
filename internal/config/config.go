package config

import (
	"log"
	"os"
)

type Mode string

const (
	ModeLocal Mode = "local"
	ModeGCP   Mode = "gcp"
)

type Config struct {
	Mode Mode

	Port string

	GCPProjectID string
	GCPLocation  string
	ModelName    string

	StorageBackend string // "memory" o "firestore"
	UseMockLLM     bool   // true = use mock even on GCP
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getBoolEnv(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	if v == "1" || v == "true" || v == "TRUE" {
		return true
	}
	return false
}

// Load reads all env vars and builds the config
func Load() *Config {
	modeStr := getEnv("FARUM_MODE", "local")
	var mode Mode
	switch modeStr {
	case "gcp":
		mode = ModeGCP
	default:
		mode = ModeLocal
	}

	cfg := &Config{
		Mode: mode,

		Port: getEnv("FARUM_PORT", "8080"),

		GCPProjectID: getEnv("FARUM_GCP_PROJECT", ""),
		GCPLocation:  getEnv("FARUM_GCP_LOCATION", "us-central1"),
		ModelName:    getEnv("FARUM_MODEL_NAME", "gemini-2.5-flash-lite"),

		StorageBackend: getEnv("FARUM_STORAGE_BACKEND", "memory"),
		UseMockLLM:     getBoolEnv("FARUM_USE_MOCK_LLM", mode == ModeLocal),
	}

	// Minimal validation in GCP mode
	if cfg.Mode == ModeGCP && cfg.GCPProjectID == "" {
		log.Fatal("FARUM_GCP_PROJECT must be set in gcp mode")
	}

	return cfg
}
