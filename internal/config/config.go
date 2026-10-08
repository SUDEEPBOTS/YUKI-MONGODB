/**
 * =================================================================================================
 *   __     __ _    _  _  __ _____     __  __  ____   _   _   ____   ____   ____   ____  
 *   \ \   / /| |  | || |/ /|_   _|   |  \/  |/ __ \ | \ | | / ___| / __ \ |  _ \ | __ ) 
 *    \ \ / / | |  | || ' /   | |     | |\/| || |  | ||  \| || |  _ | |  | || | | ||  _ \ 
 *     \ V /  | |__| || . \  _| |_    | |  | || |__| || |\  || |_| || |__| || |_| || |_) |
 *      \_/    \____/ |_|\_\|_____|   |_|  |_| \____/ |_| \_| \____| \____/ |____/ |____/  
 *
 * =================================================================================================
 * Project       : YUKI-MONGODB
 * Official Repo : https://github.com/SUDEEPBOTS/YUKI-MONGODB
 * Description   : Enterprise Configuration Parser & Environment Manager
 * Maintainer    : SUDEEPBOTS <https://github.com/SUDEEPBOTS>
 *
 * Copyright (c) 2026 SUDEEPBOTS. All rights reserved.
 * Licensed under the MIT License (https://opensource.org/licenses/MIT)
 * =================================================================================================
 */

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
	MongoUser       string
	MongoPass       string
	Domain          string
	SessionString   string
	AppID           int32
	AppHash         string
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

	appIDInt, _ := strconv.Atoi(getEnv("API_ID", "38674666"))

	if syncInterval < 1 {
		syncInterval = 5
	}

	cfg := &Config{
		Port:            port,
		MongoPort:       mongoPort,
		CacheSizeGB:     cacheSize,
		MongoUser:       getEnv("MONGO_USER", ""),
		MongoPass:       getEnv("MONGO_PASS", ""),
		Domain:          getEnv("DOMAIN", "mongo.yukiapi.site"),
		SessionString:   getEnv("SESSION_STRING", ""),
		AppID:           int32(appIDInt),
		AppHash:         getEnv("API_HASH", "b4f0fbf8fb560c4bc9e7b9f3698e474c"),
		BotToken:        os.Getenv("BOT_TOKEN"),
		ChannelID:       channelID,
		TunnelToken:     os.Getenv("TUNNEL_TOKEN"),
		SyncIntervalMin: syncInterval,
		DataDir:         getEnv("DATA_DIR", "/data/db"),
	}

	authType := "None"
	if cfg.SessionString != "" {
		authType = "MTProto Worker Session (Pure Socket)"
	} else if cfg.BotToken != "" {
		authType = "Telegram Bot Token"
	}

	log.Printf("[Config] Render Port: %d | Mongo Port: %d | Auth: %s | Channel: %d | Sync: %d min",
		cfg.Port, cfg.MongoPort, authType, cfg.ChannelID, cfg.SyncIntervalMin)

	return cfg
}

func (c *Config) HasMongoAuth() bool {
	return c.MongoUser != "" && c.MongoPass != ""
}

func (c *Config) IsTelegramConfigured() bool {
	return (c.SessionString != "" || c.BotToken != "") && c.ChannelID != 0
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
