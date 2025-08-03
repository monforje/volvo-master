package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"volvomaster/internal/bot"
	"volvomaster/internal/config"
	"volvomaster/internal/database"
	"volvomaster/internal/services"

	"github.com/joho/godotenv"
)

func main() {
	// Загружаем переменные окружения
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используем системные переменные")
	}

	// Инициализация конфигурации
	cfg := config.Load()

	// Подключение к MongoDB
	db, err := database.Connect(cfg.MongoURI)
	if err != nil {
		log.Fatal("Ошибка подключения к MongoDB:", err)
	}
	defer db.Disconnect(context.Background())

	// Инициализация сервисов
	dbService := services.NewDatabaseService(db)

	// Создание и запуск бота
	telegramBot, err := bot.NewBot(cfg.TelegramToken, dbService)
	if err != nil {
		log.Fatal("Ошибка создания бота:", err)
	}

	// Запуск бота в отдельной горутине
	go func() {
		log.Println("Бот запущен...")
		telegramBot.Start()
	}()

	// Ожидание сигнала завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Завершение работы бота...")
	telegramBot.Stop()
}
