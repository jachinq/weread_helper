package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	APIKey       string
	SkillVersion string
	ListenAddr   string
	GatewayURL   string
	DatabasePath string
	WebDir       string
	SyncInterval time.Duration
}

func Load() Config {
	_ = godotenv.Load()
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	interval := getenv("SYNC_INTERVAL", "6h")
	d, err := time.ParseDuration(interval)
	if err != nil || d <= 0 {
		d = 6 * time.Hour
	}

	cfg := Config{
		APIKey:       os.Getenv("WEREAD_API_KEY"),
		SkillVersion: getenv("SKILL_VERSION", "1.0.4"),
		ListenAddr:   getenv("LISTEN_ADDR", ":8080"),
		GatewayURL:   getenv("GATEWAY_URL", "https://i.weread.qq.com/api/agent/gateway"),
		DatabasePath: getenv("DATABASE_PATH", filepath.Join("data", "weread.db")),
		WebDir:       os.Getenv("WEB_DIR"),
		SyncInterval: d,
	}
	return cfg
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
