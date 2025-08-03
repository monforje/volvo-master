package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"volvomaster/internal/models"
	"volvomaster/internal/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CallbackHandlers struct {
	api       *tgbotapi.BotAPI
	dbService *services.DatabaseService
}

func NewCallbackHandlers(api *tgbotapi.BotAPI, dbService *services.DatabaseService) *CallbackHandlers {
	return &CallbackHandlers{
		api:       api,
		dbService: dbService,
	}
}

func (c *CallbackHandlers) HandleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	ctx := context.Background()
	userID := callback.From.ID
	data := callback.Data

	// Отвечаем на callback
	c.api.Request(tgbotapi.NewCallback(callback.ID, ""))

	session, err := c.dbService.GetUserSession(ctx, userID)
	if err != nil {
		log.Printf("Ошибка получения сессии: %v", err)
		return
	}

	request, err := c.dbService.GetServiceRequest(ctx, session.RequestID)
	if err != nil {
		log.Printf("Ошибка получения заявки: %v", err)
		return
	}

	// Обрабатываем разные типы callback
	if strings.HasPrefix(data, "engine_") {
		c.handleEngineType(callback, session, request, data)
	} else if strings.HasPrefix(data, "appeared_") {
		c.handleProblemAppeared(callback, session, request, data)
	} else if strings.HasPrefix(data, "freq_") {
		c.handleProblemFrequency(callback, session, request, data)
	} else if strings.HasPrefix(data, "date_") {
		c.handleDateSelection(callback, session, request, data)
	}
}

func (c *CallbackHandlers) handleEngineType(callback *tgbotapi.CallbackQuery, session *models.UserSession, request *models.ServiceRequest, data string) {
	ctx := context.Background()
	chatID := callback.Message.Chat.ID

	engineType := strings.TrimPrefix(data, "engine_")
	request.EngineType = engineType
	session.Data["step"] = "engine_volume"

	if err := c.dbService.SaveServiceRequest(ctx, request); err != nil {
		log.Printf("Ошибка сохранения заявки: %v", err)
		c.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
		return
	}
	if err := c.dbService.SaveUserSession(ctx, session); err != nil {
		log.Printf("Ошибка сохранения сессии: %v", err)
		c.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
		return
	}

	// Удаляем клавиатуру
	edit := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID,
		fmt.Sprintf("✅ Выбран тип двигателя: %s", engineType))
	c.api.Send(edit)

	c.sendMessage(chatID, "Объем двигателя (если знаете, например: 2.0, 2.4, 3.0). Если не знаете, напишите 'не знаю':")
}

func (c *CallbackHandlers) handleProblemAppeared(callback *tgbotapi.CallbackQuery, session *models.UserSession, request *models.ServiceRequest, data string) {
	ctx := context.Background()
	chatID := callback.Message.Chat.ID

	appeared := strings.TrimPrefix(data, "appeared_")
	request.ProblemFirstAppeared = appeared
	session.Data["step"] = "problem_frequency"

	if err := c.dbService.SaveServiceRequest(ctx, request); err != nil {
		log.Printf("Ошибка сохранения заявки: %v", err)
		c.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
		return
	}
	if err := c.dbService.SaveUserSession(ctx, session); err != nil {
		log.Printf("Ошибка сохранения сессии: %v", err)
		c.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
		return
	}

	// Удаляем клавиатуру
	edit := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID,
		fmt.Sprintf("✅ Проблема появилась: %s", appeared))
	c.api.Send(edit)

	// Создаем клавиатуру для частоты проблемы
	keyboard := tgbotapi.NewInlineKeyboardMarkup()
	for _, freq := range models.ProblemFrequencies {
		// Используем короткие идентификаторы без пробелов
		var callbackData string
		switch freq {
		case "Постоянно":
			callbackData = "freq_always"
		case "Периодически":
			callbackData = "freq_periodic"
		case "Только при определенных условиях":
			callbackData = "freq_conditional"
		default:
			callbackData = "freq_other"
		}

		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(freq, callbackData),
		)
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
	}

	msg := tgbotapi.NewMessage(chatID, "Проблема проявляется постоянно или периодически?")
	msg.ReplyMarkup = keyboard
	c.api.Send(msg)
}

func (c *CallbackHandlers) handleProblemFrequency(callback *tgbotapi.CallbackQuery, session *models.UserSession, request *models.ServiceRequest, data string) {
	ctx := context.Background()
	chatID := callback.Message.Chat.ID

	// Преобразуем callback данные обратно в читаемый текст
	var frequency string
	switch data {
	case "freq_always":
		frequency = "Постоянно"
	case "freq_periodic":
		frequency = "Периодически"
	case "freq_conditional":
		frequency = "Только при определенных условиях"
	default:
		frequency = "Не указано"
	}
	request.ProblemFrequency = frequency
	session.Data["step"] = "safety_impact"

	if err := c.dbService.SaveServiceRequest(ctx, request); err != nil {
		log.Printf("Ошибка сохранения заявки: %v", err)
		c.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
		return
	}
	if err := c.dbService.SaveUserSession(ctx, session); err != nil {
		log.Printf("Ошибка сохранения сессии: %v", err)
		c.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
		return
	}

	// Удаляем клавиатуру
	edit := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID,
		fmt.Sprintf("✅ Частота проблемы: %s", frequency))
	c.api.Send(edit)

	c.sendMessage(chatID, `Влияет ли это на движение или безопасность?

Например: "машина не заводится", "перестали работать тормоза", "не влияет на безопасность"`)
}

func (c *CallbackHandlers) handleDateSelection(callback *tgbotapi.CallbackQuery, session *models.UserSession, request *models.ServiceRequest, data string) {
	ctx := context.Background()
	chatID := callback.Message.Chat.ID
	userID := callback.From.ID

	timestampStr := strings.TrimPrefix(data, "date_")
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		log.Printf("Ошибка парсинга времени: %v", err)
		return
	}

	selectedDate := time.Unix(timestamp, 0)
	request.AppointmentDate = selectedDate
	request.Status = "completed"
	session.Stage = models.StageCompleted

	if err := c.dbService.SaveServiceRequest(ctx, request); err != nil {
		log.Printf("Ошибка сохранения заявки: %v", err)
		c.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
		return
	}
	if err := c.dbService.SaveUserSession(ctx, session); err != nil {
		log.Printf("Ошибка сохранения сессии: %v", err)
		c.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
		return
	}

	// Удаляем клавиатуру
	edit := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID,
		fmt.Sprintf("✅ Выбрана дата: %s", selectedDate.Format("02.01.2006 15:04")))
	c.api.Send(edit)

	// Календарь отключен - заявка сохраняется только в базе данных

	// Отправляем итоговое сообщение
	c.sendCompletionMessage(chatID, request)

	// Удаляем сессию
	c.dbService.DeleteUserSession(ctx, userID)
}

func (c *CallbackHandlers) sendCompletionMessage(chatID int64, request *models.ServiceRequest) {
	completionText := fmt.Sprintf(`
🎉 ЗАЯВКА УСПЕШНО СОЗДАНА!

📋 Резюме вашей заявки:

👤 Клиент: %s
📞 Контакт: %s

🚗 Автомобиль:
• Модель: %s %s
• Тип двигателя: %s
• Объем: %s
• Пробег: %s км

🔧 Проблема: %s
📅 Дата визита: %s

✅ Ваша заявка создана и сохранена.
📞 Мы свяжемся с вами для подтверждения записи.

Спасибо за обращение! 🚗💙

Для создания новой заявки используйте /start`,
		request.Name,
		request.Contact,
		request.VolvoModel,
		request.Year,
		request.EngineType,
		request.EngineVolume,
		request.Mileage,
		request.Problem,
		request.AppointmentDate.Format("02.01.2006 15:04"))

	c.sendMessage(chatID, completionText)
}

func (c *CallbackHandlers) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	_, err := c.api.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения пользователю %d: %v", chatID, err)
	}
}
