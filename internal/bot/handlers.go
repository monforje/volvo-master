package bot

import (
	"context"
	"log"
	"strings"

	"volvomaster/internal/models"
	"volvomaster/internal/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handlers struct {
	api       *tgbotapi.BotAPI
	dbService *services.DatabaseService
	stages    *StageHandlers
	callbacks *CallbackHandlers
}

func NewHandlers(api *tgbotapi.BotAPI, dbService *services.DatabaseService) *Handlers {
	return &Handlers{
		api:       api,
		dbService: dbService,
		stages:    NewStageHandlers(api, dbService),
		callbacks: NewCallbackHandlers(api, dbService),
	}
}

func (h *Handlers) HandleMessage(message *tgbotapi.Message) {
	ctx := context.Background()
	userID := message.From.ID
	chatID := message.Chat.ID

	log.Printf("Получено сообщение от %d: %s", userID, message.Text)

	// Получаем сессию пользователя
	session, err := h.dbService.GetUserSession(ctx, userID)
	if err != nil {
		log.Printf("Ошибка получения сессии: %v", err)
		h.sendMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		return
	}

	session.ChatID = chatID

	// Обрабатываем команды
	if strings.HasPrefix(message.Text, "/") {
		h.handleCommand(message, session)
		return
	}

	// Обрабатываем сообщение в зависимости от этапа
	switch session.Stage {
	case models.StageStart:
		h.stages.StartNewRequest(session)
	case models.StagePersonalInfo:
		h.stages.HandlePersonalInfo(message, session)
	case models.StageCarInfo:
		h.stages.HandleCarInfo(message, session)
	case models.StageProblemInfo:
		h.stages.HandleProblemInfo(message, session)
	default:
		h.sendMessage(chatID, "Произошла ошибка. Начните заново с команды /start")
	}
}

func (h *Handlers) handleCommand(message *tgbotapi.Message, session *models.UserSession) {
	ctx := context.Background()
	chatID := message.Chat.ID

	switch message.Text {
	case "/start":
		h.stages.StartNewRequest(session)
	case "/cancel":
		h.dbService.DeleteUserSession(ctx, session.UserID)
		h.sendMessage(chatID, "Заявка отменена. Для создания новой заявки используйте /start")
	case "/help":
		helpText := `
🛠 Бот для записи на обслуживание Volvo

Доступные команды:
/start - Начать новую заявку
/cancel - Отменить текущую заявку
/help - Показать эту справку

Бот поможет вам записаться на обслуживание вашего автомобиля Volvo.
Просто следуйте инструкциям и отвечайте на вопросы.
		`
		h.sendMessage(chatID, helpText)
	}
}

func (h *Handlers) HandleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	h.callbacks.HandleCallbackQuery(callback)
}

func (h *Handlers) HandleChatMemberUpdate(update *tgbotapi.ChatMemberUpdated) {
	chatID := update.Chat.ID
	userID := update.From.ID
	newStatus := update.NewChatMember.Status
	oldStatus := update.OldChatMember.Status

	log.Printf("Изменение статуса бота в чате %d: %s -> %s", chatID, oldStatus, newStatus)

	// Если бота заблокировали
	if newStatus == "kicked" {
		log.Printf("Бот заблокирован пользователем %d", userID)
		ctx := context.Background()
		// Очищаем данные пользователя
		h.dbService.DeleteUserSession(ctx, userID)
		log.Printf("Данные пользователя %d очищены", userID)
	}

	// Если бота разблокировали
	if oldStatus == "kicked" && newStatus == "member" {
		log.Printf("Бот разблокирован пользователем %d", userID)
		// Можно отправить приветственное сообщение
		h.sendMessage(chatID, "👋 Спасибо, что разблокировали меня! Используйте /start для начала работы.")
	}
}

func (h *Handlers) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	_, err := h.api.Send(msg)
	if err != nil {
		// Проверяем тип ошибки
		if strings.Contains(err.Error(), "bot was blocked by the user") {
			log.Printf("Пользователь %d заблокировал бота", chatID)
			// Можно добавить логику для очистки данных пользователя
			ctx := context.Background()
			h.dbService.DeleteUserSession(ctx, chatID)
		} else if strings.Contains(err.Error(), "chat not found") {
			log.Printf("Чат %d не найден", chatID)
		} else if strings.Contains(err.Error(), "user is deactivated") {
			log.Printf("Пользователь %d деактивирован", chatID)
		} else {
			log.Printf("Ошибка отправки сообщения пользователю %d: %v", chatID, err)
		}
	}
}
