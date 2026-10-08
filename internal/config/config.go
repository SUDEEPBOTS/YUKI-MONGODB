package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	Port            int
	MongoPort       int
	CacheSizeGB     float64
	BotToken        string
	ChannelID       int64
	TunnelToken     string
	SyncIntervalMin int
	DataDir         string
}

func LoadConfig() *Config {
	port, _ := strconv.Atoi(getEnv("PORT", "10000"))
	mongoPort, _ := strconv.Atoi(getEnv("MONGO_PORT", "27017"))
	cacheSize, _ := strconv.ParseFloat(getEnv("CACHE_SIZE_GB", "0.25"), 64)
	channelID, _ := strconv.ParseInt(getEnv("CHANNEL_ID", "0"), 10, 64)
	syncInterval, _ := strconv.Atoi(getEnv("SYNC_INTERVAL_MIN", "5"))

	if syncInterval < 1 {
		syncInterval = 5
	}

	cfg := &Config{
		Port:            port,
		MongoPort:       mongoPort,
		CacheSizeGB:     cacheSize,
		BotToken:        os.Getenv("BOT_TOKEN"),
		ChannelID:       channelID,
		TunnelToken:     os.Getenv("TUNNEL_TOKEN"),
		SyncIntervalMin: syncInterval,
		DataDir:         getEnv("DATA_DIR", "/data/db"),
	}

	log.Printf("[Config] Render HTTP Port: %d | Mongo Port: %d | WiredTiger Cap: %.2f GB | Sync Interval: %d min",
		cfg.Port, cfg.MongoPort, cfg.CacheSizeGB, cfg.SyncIntervalMin)

	return cfg
}

func (c *Config) IsTelegramConfigured() bool {
	return c.BotToken != "" && c.ChannelID != 0
}

func (c *Config) IsTunnelConfigured() bool {
	return c.TunnelToken != ""
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
