package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"volvomaster/internal/bot"
	"volvomaster/internal/config"
	"volvomaster/internal/database"
	"volvomaster/internal/logger"
	"volvomaster/internal/services"

	"github.com/joho/godotenv"
)

func main() {
	// Инициализация логгера
	logger := logger.New()

	// Загружаем переменные окружения
	if err := godotenv.Load(); err != nil {
		logger.Info("Файл .env не найден, используем системные переменные")
	}

	// Инициализация конфигурации
	cfg := config.Load()

	// Подключение к MongoDB
	db, err := database.Connect(cfg.MongoURI)
	if err != nil {
		logger.Fatal("Ошибка подключения к MongoDB: %v", err)
	}
	defer db.Disconnect(context.Background())

	// Инициализация сервисов
	dbService := services.NewDatabaseService(db)

	// Создание и запуск бота
	telegramBot, err := bot.NewBot(cfg.TelegramToken, dbService)
	if err != nil {
		logger.Fatal("Ошибка создания бота: %v", err)
	}

	// Запуск бота в отдельной горутине
	go func() {
		logger.Info("Бот запущен...")
		telegramBot.Start()
	}()

	// Ожидание сигнала завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Завершение работы бота...")
	telegramBot.Stop()
}
