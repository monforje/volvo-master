package config

import "os"

type Config struct {
	TelegramToken string
	MongoURI      string
}

func Load() *Config {
	token := getEnv("TELEGRAM_BOT_TOKEN", "")
	if token == "" {
		panic("TELEGRAM_BOT_TOKEN не установлен в переменных окружения")
	}

	return &Config{
		TelegramToken: token,
		MongoURI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
