package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"volvomaster/internal/logger"
	"volvomaster/internal/models"
	"volvomaster/internal/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Bot struct {
	api       *tgbotapi.BotAPI
	dbService *services.DatabaseService
	logger    *logger.Logger
	stopChan  chan struct{}
	isRunning bool
}

func NewBot(token string, dbService *services.DatabaseService) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания бота: %w", err)
	}

	return &Bot{
		api:       api,
		dbService: dbService,
		logger:    logger.New(),
		stopChan:  make(chan struct{}),
	}, nil
}

func (b *Bot) Start() {
	b.isRunning = true
	b.logger.Info("Бот запущен")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case update := <-updates:
			if !b.isRunning {
				return
			}
			go b.handleUpdate(update)
		case <-b.stopChan:
			return
		}
	}
}

func (b *Bot) Stop() {
	b.isRunning = false
	close(b.stopChan)
	b.logger.Info("Бот остановлен")
}

func (b *Bot) handleUpdate(update tgbotapi.Update) {
	ctx := context.Background()

	// Обрабатываем callback-запросы
	if update.CallbackQuery != nil {
		b.handleCallbackQuery(update.CallbackQuery)
		return
	}

	if update.Message == nil {
		return
	}

	message := update.Message
	userID := message.From.ID
	chatID := message.Chat.ID

	b.logger.Info("Получено сообщение от пользователя %d: %s", userID, message.Text)

	// Сохраняем или обновляем информацию о пользователе
	user := &models.User{
		UserID:    userID,
		ChatID:    chatID,
		Username:  message.From.UserName,
		FirstName: message.From.FirstName,
		LastName:  message.From.LastName,
		UpdatedAt: time.Now(),
	}

	// Если пользователь отправил контакт, сохраняем номер телефона
	if message.Contact != nil && message.Contact.UserID == userID {
		user.Phone = message.Contact.PhoneNumber
	}

	if err := b.dbService.SaveUser(ctx, user); err != nil {
		b.logger.Error("Ошибка сохранения пользователя: %v", err)
	}

	// Получаем сессию пользователя
	session, err := b.dbService.GetUserSession(ctx, userID)
	if err != nil {
		b.logger.Error("Ошибка получения сессии: %v", err)
		b.sendMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		return
	}

	// Устанавливаем chat_id в сессии, если он не установлен
	if session.ChatID == 0 {
		session.ChatID = chatID
		if err := b.dbService.SaveUserSession(ctx, session); err != nil {
			b.logger.Error("Ошибка сохранения сессии: %v", err)
		}
	}

	// Обрабатываем команды
	if message.IsCommand() {
		b.handleCommand(ctx, message, session)
		return
	}

	// Обрабатываем сообщения в зависимости от этапа
	b.handleStage(ctx, message, session)
}

func (b *Bot) handleCommand(ctx context.Context, message *tgbotapi.Message, session *models.UserSession) {
	chatID := message.Chat.ID

	switch message.Command() {
	case "start":
		// Сбрасываем сессию и начинаем заново
		session.Stage = models.StagePersonalInfo
		session.Data = make(map[string]interface{})
		session.RequestID = primitive.NilObjectID

		if err := b.dbService.SaveUserSession(ctx, session); err != nil {
			b.logger.Error("Ошибка сохранения сессии: %v", err)
		}

		welcomeText := `Добро пожаловать в сервисный центр Volvo! 🚗

Я помогу вам записаться на обслуживание вашего автомобиля.

Для начала заполним заявку. Начнем с ваших контактных данных.

Как вас зовут?`

		b.sendMessage(chatID, welcomeText)

	case "cancel":
		// Отменяем текущую заявку
		if !session.RequestID.IsZero() {
			request, err := b.dbService.GetServiceRequest(ctx, session.RequestID)
			if err == nil {
				request.Status = "cancelled"
				b.dbService.SaveServiceRequest(ctx, request)
			}
		}

		session.Stage = models.StageStart
		session.Data = make(map[string]interface{})
		session.RequestID = primitive.NilObjectID
		b.dbService.SaveUserSession(ctx, session)

		b.sendMessage(chatID, "Заявка отменена. Нажмите /start для создания новой заявки.")

	case "help":
		helpText := `Доступные команды:
/start - Начать новую заявку
/cancel - Отменить текущую заявку
/help - Показать эту справку`
		b.sendMessage(chatID, helpText)
	}
}

func (b *Bot) handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	ctx := context.Background()
	userID := callback.From.ID
	data := callback.Data

	b.logger.Info("Получен callback от пользователя %d: %s", userID, data)

	// Получаем сессию пользователя
	session, err := b.dbService.GetUserSession(ctx, userID)
	if err != nil {
		b.logger.Error("Ошибка получения сессии: %v", err)
		b.answerCallback(callback.ID, "Произошла ошибка. Попробуйте позже.")
		return
	}

	// Обрабатываем callback в зависимости от типа
	if strings.HasPrefix(data, "date_") {
		b.handleDateSelection(ctx, callback, session)
	} else if strings.HasPrefix(data, "time_") {
		b.handleTimeSelection(ctx, callback, session)
	} else if strings.HasPrefix(data, "engine_") {
		b.handleEngineTypeSelection(ctx, callback, session)
	} else if strings.HasPrefix(data, "appeared_") {
		b.handleProblemAppearedSelection(ctx, callback, session)
	} else if strings.HasPrefix(data, "frequency_") {
		b.handleProblemFrequencySelection(ctx, callback, session)
	} else {
		b.answerCallback(callback.ID, "Неизвестный callback")
	}
}

func (b *Bot) handleStage(ctx context.Context, message *tgbotapi.Message, session *models.UserSession) {
	chatID := message.Chat.ID

	switch session.Stage {
	case models.StageStart:
		// Начинаем с первого этапа
		session.Stage = models.StagePersonalInfo
		b.sendMessage(chatID, "Как вас зовут?")
		b.dbService.SaveUserSession(ctx, session)

	case models.StagePersonalInfo:
		b.handlePersonalInfoStage(ctx, message, session)

	case models.StageCarInfo:
		b.handleCarInfoStage(ctx, message, session)

	case models.StageProblemInfo:
		b.handleProblemInfoStage(ctx, message, session)

	case models.StageDateSelection:
		b.handleDateSelectionStage(ctx, message, session)

	case models.StageCompleted:
		b.sendMessage(chatID, "Ваша заявка уже завершена. Нажмите /start для создания новой заявки.")
	}
}

func (b *Bot) handlePersonalInfoStage(ctx context.Context, message *tgbotapi.Message, session *models.UserSession) {
	chatID := message.Chat.ID
	text := message.Text

	// Определяем, какое поле заполняем
	if session.Data["name"] == nil {
		// Сохраняем имя
		session.Data["name"] = text

		// Запрашиваем контактную информацию
		contactText := `Укажите номер телефона или Telegram для связи:`
		b.sendMessage(chatID, contactText)

		// Сохраняем сессию после первого ответа
		b.dbService.SaveUserSession(ctx, session)
	} else {
		// Сохраняем контакт
		contact := text

		// Если пользователь ввел @username, сохраняем его
		if strings.HasPrefix(contact, "@") {
			session.Data["contact"] = contact
		} else {
			// Если это номер телефона, сохраняем его
			session.Data["contact"] = contact
		}

		// Обновляем информацию о пользователе
		user, err := b.dbService.GetUser(ctx, message.From.ID)
		if err == nil {
			// Если пользователь указал номер телефона, сохраняем его
			if !strings.HasPrefix(contact, "@") {
				user.Phone = contact
			}
			// Если пользователь указал Telegram username, сохраняем его
			if strings.HasPrefix(contact, "@") {
				user.Telegram = contact
			}
			if err := b.dbService.SaveUser(ctx, user); err != nil {
				b.logger.Error("Ошибка обновления пользователя: %v", err)
			}
		} else {
			b.logger.Error("Ошибка получения пользователя для обновления: %v", err)
		}

		// Создаем заявку
		request := &models.ServiceRequest{
			UserID:  message.From.ID,
			ChatID:  chatID,
			Name:    session.Data["name"].(string),
			Contact: session.Data["contact"].(string),
			Stage:   models.StageCarInfo,
			Status:  "in_progress",
		}

		if err := b.dbService.SaveServiceRequest(ctx, request); err != nil {
			b.logger.Error("Ошибка сохранения заявки: %v", err)
			b.sendMessage(chatID, "Произошла ошибка. Попробуйте позже.")
			return
		}

		session.RequestID = request.ID
		session.Stage = models.StageCarInfo
		session.Data = make(map[string]interface{})

		b.dbService.SaveUserSession(ctx, session)

		// Скрываем клавиатуру
		removeKeyboard := tgbotapi.NewRemoveKeyboard(true)
		msg := tgbotapi.NewMessage(chatID, "")
		msg.ReplyMarkup = removeKeyboard
		b.api.Send(msg)

		// Переходим к информации об автомобиле
		carInfoText := `Отлично! Теперь расскажите о вашем автомобиле.

Какая у вас модель Volvo?`
		b.sendMessage(chatID, carInfoText)
	}
}

func (b *Bot) handleCarInfoStage(ctx context.Context, message *tgbotapi.Message, session *models.UserSession) {
	chatID := message.Chat.ID
	text := message.Text

	request, err := b.dbService.GetServiceRequest(ctx, session.RequestID)
	if err != nil {
		b.logger.Error("Ошибка получения заявки: %v", err)
		b.sendMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		return
	}

	// Определяем, какое поле заполняем
	if request.VolvoModel == "" {
		request.VolvoModel = text
		if err := b.dbService.SaveServiceRequest(ctx, request); err != nil {
			b.logger.Error("Ошибка сохранения заявки: %v", err)
			b.sendMessage(chatID, "Произошла ошибка. Попробуйте позже.")
			return
		}
		b.sendMessage(chatID, "Укажите год выпуска автомобиля:")
	} else if request.Year == "" {
		request.Year = text
		if err := b.dbService.SaveServiceRequest(ctx, request); err != nil {
			b.logger.Error("Ошибка сохранения заявки: %v", err)
			b.sendMessage(chatID, "Произошла ошибка. Попробуйте позже.")
			return
		}

		// Показываем типы двигателей с кнопками
		b.showEngineTypes(chatID)
	} else if request.EngineType == "" {
		// Обрабатываем выбор типа двигателя через callback
		b.sendMessage(chatID, "Пожалуйста, выберите тип двигателя из предложенных вариантов выше.")
	} else if request.EngineVolume == "" {
		request.EngineVolume = text
		if err := b.dbService.SaveServiceRequest(ctx, request); err != nil {
			b.logger.Error("Ошибка сохранения заявки: %v", err)
			b.sendMessage(chatID, "Произошла ошибка. Попробуйте позже.")
			return
		}
		b.sendMessage(chatID, "Укажите пробег автомобиля на текущий момент:")
	} else if request.Mileage == "" {
		request.Mileage = text

		// Сохраняем информацию об автомобиле
		request.Stage = models.StageProblemInfo
		if err := b.dbService.SaveServiceRequest(ctx, request); err != nil {
			b.logger.Error("Ошибка сохранения заявки: %v", err)
			b.sendMessage(chatID, "Произошла ошибка. Попробуйте позже.")
			return
		}

		session.Stage = models.StageProblemInfo
		b.dbService.SaveUserSession(ctx, session)

		// Переходим к информации о проблеме
		problemText := `Теперь расскажите о проблеме с автомобилем.

Что именно вас беспокоит или что нужно сделать?
(Например: "гремит спереди", "нужно заменить масло", "ошибка по двигателю", "не работает климат")`
		b.sendMessage(chatID, problemText)
	}
}

func (b *Bot) handleProblemInfoStage(ctx context.Context, message *tgbotapi.Message, session *models.UserSession) {
	chatID := message.Chat.ID
	text := message.Text

	request, err := b.dbService.GetServiceRequest(ctx, session.RequestID)
	if err != nil {
		b.logger.Error("Ошибка получения заявки: %v", err)
		b.sendMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		return
	}

	// Определяем, какое поле заполняем
	if request.Problem == "" {
		request.Problem = text
		if err := b.dbService.SaveServiceRequest(ctx, request); err != nil {
			b.logger.Error("Ошибка сохранения заявки: %v", err)
			b.sendMessage(chatID, "Произошла ошибка. Попробуйте позже.")
			return
		}

		// Показываем варианты когда появилась проблема с кнопками
		b.showProblemAppeared(chatID)
	} else if request.ProblemFirstAppeared == "" {
		// Обрабатываем выбор через callback
		b.sendMessage(chatID, "Пожалуйста, выберите вариант из предложенных выше.")
	} else if request.ProblemFrequency == "" {
		// Обрабатываем выбор через callback
		b.sendMessage(chatID, "Пожалуйста, выберите вариант из предложенных выше.")
	} else if request.SafetyImpact == "" {
		request.SafetyImpact = text
		if err := b.dbService.SaveServiceRequest(ctx, request); err != nil {
			b.logger.Error("Ошибка сохранения заявки: %v", err)
			b.sendMessage(chatID, "Произошла ошибка. Попробуйте позже.")
			return
		}
		b.sendMessage(chatID, "Уже предпринимались попытки ремонта или диагностики? (Если да — что делали и где?)")
	} else if request.PreviousRepairs == "" {
		request.PreviousRepairs = text
		if err := b.dbService.SaveServiceRequest(ctx, request); err != nil {
			b.logger.Error("Ошибка сохранения заявки: %v", err)
			b.sendMessage(chatID, "Произошла ошибка. Попробуйте позже.")
			return
		}
		b.sendMessage(chatID, "Меняли ли что-то недавно? (Например: \"меняли подвеску месяц назад\")")
	} else if request.RecentChanges == "" {
		request.RecentChanges = text

		// Сохраняем информацию о проблеме
		request.Stage = models.StageDateSelection
		if err := b.dbService.SaveServiceRequest(ctx, request); err != nil {
			b.logger.Error("Ошибка сохранения заявки: %v", err)
			b.sendMessage(chatID, "Произошла ошибка. Попробуйте позже.")
			return
		}

		session.Stage = models.StageDateSelection
		b.dbService.SaveUserSession(ctx, session)

		// Показываем доступные даты
		b.showAvailableDates(chatID)
	}
}

func (b *Bot) handleDateSelectionStage(ctx context.Context, message *tgbotapi.Message, session *models.UserSession) {
	chatID := message.Chat.ID

	// В этапе выбора даты обрабатываем только текстовые сообщения
	// Callback-запросы обрабатываются отдельно в handleCallbackQuery
	b.sendMessage(chatID, "Пожалуйста, выберите дату из предложенных вариантов выше.")
}

func (b *Bot) showAvailableDates(chatID int64) {
	ctx := context.Background()

	dates, err := b.dbService.GetAvailableDates(ctx)
	if err != nil {
		b.logger.Error("Ошибка получения доступных дат: %v", err)
		b.sendMessage(chatID, "Произошла ошибка при получении доступных дат.")
		return
	}

	if len(dates) == 0 {
		b.sendMessage(chatID, "К сожалению, на данный момент нет доступных дат для записи. Попробуйте позже.")
		return
	}

	// Группируем даты по дню, чтобы избежать дубликатов
	uniqueDates := make(map[string]*models.AvailableDate)
	for _, date := range dates {
		dateKey := date.Date.Format("2006-01-02")
		if _, exists := uniqueDates[dateKey]; !exists {
			uniqueDates[dateKey] = date
		}
	}

	text := "Выберите удобную дату для записи:"
	var keyboard [][]tgbotapi.InlineKeyboardButton

	for _, date := range uniqueDates {
		// Получаем правильное название дня недели
		weekday := getWeekdayName(date.Date.Weekday())
		dateText := date.Date.Format("02.01.2006") + " (" + weekday + ")"

		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				dateText,
				fmt.Sprintf("date_%s", date.ID.Hex()),
			),
		}
		keyboard = append(keyboard, row)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)

	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Ошибка отправки сообщения: %v", err)
	}
}

func (b *Bot) showTimeSlots(chatID int64, availableDate *models.AvailableDate) {
	text := fmt.Sprintf("Выберите время для записи на %s:\n\n",
		availableDate.Date.Format("02.01.2006"))

	var keyboard [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	for i, slot := range availableDate.TimeSlots {
		if !slot.IsBooked {
			button := tgbotapi.NewInlineKeyboardButtonData(
				slot.Time,
				fmt.Sprintf("time_%s_%s", availableDate.ID.Hex(), slot.Time),
			)
			row = append(row, button)

			// Размещаем по 3 кнопки в ряд
			if len(row) == 3 || i == len(availableDate.TimeSlots)-1 {
				keyboard = append(keyboard, row)
				row = []tgbotapi.InlineKeyboardButton{}
			}
		}
	}

	if len(keyboard) == 0 {
		b.sendMessage(chatID, "К сожалению, на эту дату нет свободного времени.")
		return
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)

	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Ошибка отправки сообщения: %v", err)
	}
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Ошибка отправки сообщения: %v", err)
	}
}

func (b *Bot) showEngineTypes(chatID int64) {
	text := "Выберите тип двигателя:"
	var keyboard [][]tgbotapi.InlineKeyboardButton

	for _, engineType := range models.EngineTypes {
		key := models.EngineTypeKeys[engineType]
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				engineType,
				fmt.Sprintf("engine_%s", key),
			),
		}
		keyboard = append(keyboard, row)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)

	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Ошибка отправки сообщения: %v", err)
	}
}

func (b *Bot) showProblemAppeared(chatID int64) {
	text := "Когда впервые появилась проблема?"
	var keyboard [][]tgbotapi.InlineKeyboardButton

	for _, option := range models.ProblemAppeared {
		key := models.ProblemAppearedKeys[option]
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				option,
				fmt.Sprintf("appeared_%s", key),
			),
		}
		keyboard = append(keyboard, row)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)

	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Ошибка отправки сообщения: %v", err)
	}
}

func (b *Bot) showProblemFrequency(chatID int64) {
	text := "Проблема проявляется постоянно или периодически?"
	var keyboard [][]tgbotapi.InlineKeyboardButton

	for _, option := range models.ProblemFrequencies {
		key := models.ProblemFrequencyKeys[option]
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				option,
				fmt.Sprintf("frequency_%s", key),
			),
		}
		keyboard = append(keyboard, row)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)

	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Ошибка отправки сообщения: %v", err)
	}
}

func (b *Bot) handleDateSelection(ctx context.Context, callback *tgbotapi.CallbackQuery, session *models.UserSession) {
	chatID := callback.Message.Chat.ID
	data := callback.Data

	dateID := strings.TrimPrefix(data, "date_")
	if objectID, err := primitive.ObjectIDFromHex(dateID); err == nil {
		// Получаем выбранную дату
		availableDate, err := b.dbService.GetAvailableDateByID(ctx, objectID)
		if err != nil {
			b.logger.Error("Ошибка получения даты: %v", err)
			b.answerCallback(callback.ID, "Произошла ошибка. Попробуйте позже.")
			return
		}

		// Показываем временные слоты
		b.showTimeSlots(chatID, availableDate)
		b.answerCallback(callback.ID, "")
	} else {
		b.answerCallback(callback.ID, "Неверный формат даты")
	}
}

func (b *Bot) handleTimeSelection(ctx context.Context, callback *tgbotapi.CallbackQuery, session *models.UserSession) {
	chatID := callback.Message.Chat.ID
	data := callback.Data

	timeSlot := strings.TrimPrefix(data, "time_")
	parts := strings.Split(timeSlot, "_")
	if len(parts) == 2 {
		dateID := parts[0]
		timeStr := parts[1]

		if objectID, err := primitive.ObjectIDFromHex(dateID); err == nil {
			// Завершаем заявку
			request, err := b.dbService.GetServiceRequest(ctx, session.RequestID)
			if err != nil {
				b.logger.Error("Ошибка получения заявки: %v", err)
				b.answerCallback(callback.ID, "Произошла ошибка. Попробуйте позже.")
				return
			}

			// Парсим дату и время
			availableDate, err := b.dbService.GetAvailableDateByID(ctx, objectID)
			if err != nil {
				b.logger.Error("Ошибка получения даты: %v", err)
				b.answerCallback(callback.ID, "Произошла ошибка. Попробуйте позже.")
				return
			}

			// Создаем полную дату и время
			appointmentTime := time.Date(
				availableDate.Date.Year(),
				availableDate.Date.Month(),
				availableDate.Date.Day(),
				0, 0, 0, 0, availableDate.Date.Location(),
			)

			// Парсим время
			if t, err := time.Parse("15:04", timeStr); err == nil {
				appointmentTime = appointmentTime.Add(time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute)
			}

			request.AppointmentDate = appointmentTime
			request.Stage = models.StageCompleted
			request.Status = "completed"

			if err := b.dbService.SaveServiceRequest(ctx, request); err != nil {
				b.logger.Error("Ошибка сохранения заявки: %v", err)
				b.answerCallback(callback.ID, "Произошла ошибка. Попробуйте позже.")
				return
			}

			session.Stage = models.StageCompleted
			b.dbService.SaveUserSession(ctx, session)

			// Отправляем подтверждение
			confirmationText := fmt.Sprintf(`✅ Заявка успешно создана!

📋 Информация о заявке:
👤 Имя: %s
📞 Контакт: %s
🚗 Модель: %s %s
🔧 Проблема: %s
📅 Дата записи: %s

Мы свяжемся с вами для подтверждения записи.`,
				request.Name, request.Contact, request.VolvoModel, request.Year,
				request.Problem, appointmentTime.Format("02.01.2006 в 15:04"))

			b.sendMessage(chatID, confirmationText)
			b.answerCallback(callback.ID, "Заявка создана успешно!")
		} else {
			b.answerCallback(callback.ID, "Неверный формат даты")
		}
	} else {
		b.answerCallback(callback.ID, "Неверный формат времени")
	}
}

func (b *Bot) handleEngineTypeSelection(ctx context.Context, callback *tgbotapi.CallbackQuery, session *models.UserSession) {
	chatID := callback.Message.Chat.ID
	data := callback.Data

	engineKey := strings.TrimPrefix(data, "engine_")
	engineType := models.EngineTypeValues[engineKey]

	request, err := b.dbService.GetServiceRequest(ctx, session.RequestID)
	if err != nil {
		b.logger.Error("Ошибка получения заявки: %v", err)
		b.answerCallback(callback.ID, "Произошла ошибка. Попробуйте позже.")
		return
	}

	request.EngineType = engineType
	if err := b.dbService.SaveServiceRequest(ctx, request); err != nil {
		b.logger.Error("Ошибка сохранения заявки: %v", err)
		b.answerCallback(callback.ID, "Произошла ошибка. Попробуйте позже.")
		return
	}

	b.sendMessage(chatID, "Укажите объем двигателя (если знаете):")
	b.answerCallback(callback.ID, "")
}

func (b *Bot) handleProblemAppearedSelection(ctx context.Context, callback *tgbotapi.CallbackQuery, session *models.UserSession) {
	chatID := callback.Message.Chat.ID
	data := callback.Data

	appearedKey := strings.TrimPrefix(data, "appeared_")
	appeared := models.ProblemAppearedValues[appearedKey]

	request, err := b.dbService.GetServiceRequest(ctx, session.RequestID)
	if err != nil {
		b.logger.Error("Ошибка получения заявки: %v", err)
		b.answerCallback(callback.ID, "Произошла ошибка. Попробуйте позже.")
		return
	}

	request.ProblemFirstAppeared = appeared
	if err := b.dbService.SaveServiceRequest(ctx, request); err != nil {
		b.logger.Error("Ошибка сохранения заявки: %v", err)
		b.answerCallback(callback.ID, "Произошла ошибка. Попробуйте позже.")
		return
	}

	b.showProblemFrequency(chatID)
	b.answerCallback(callback.ID, "")
}

func (b *Bot) handleProblemFrequencySelection(ctx context.Context, callback *tgbotapi.CallbackQuery, session *models.UserSession) {
	chatID := callback.Message.Chat.ID
	data := callback.Data

	frequencyKey := strings.TrimPrefix(data, "frequency_")
	frequency := models.ProblemFrequencyValues[frequencyKey]

	request, err := b.dbService.GetServiceRequest(ctx, session.RequestID)
	if err != nil {
		b.logger.Error("Ошибка получения заявки: %v", err)
		b.answerCallback(callback.ID, "Произошла ошибка. Попробуйте позже.")
		return
	}

	request.ProblemFrequency = frequency
	if err := b.dbService.SaveServiceRequest(ctx, request); err != nil {
		b.logger.Error("Ошибка сохранения заявки: %v", err)
		b.answerCallback(callback.ID, "Произошла ошибка. Попробуйте позже.")
		return
	}

	b.sendMessage(chatID, "Влияет ли это на движение или безопасность? (Например: \"машина не заводится\", \"перестали работать тормоза\")")
	b.answerCallback(callback.ID, "")
}

func (b *Bot) answerCallback(callbackID string, text string) {
	callback := tgbotapi.NewCallback(callbackID, text)
	if _, err := b.api.Request(callback); err != nil {
		b.logger.Error("Ошибка ответа на callback: %v", err)
	}
}

// getWeekdayName возвращает название дня недели на русском языке
func getWeekdayName(weekday time.Weekday) string {
	weekdays := map[time.Weekday]string{
		time.Monday:    "понедельник",
		time.Tuesday:   "вторник",
		time.Wednesday: "среда",
		time.Thursday:  "четверг",
		time.Friday:    "пятница",
		time.Saturday:  "суббота",
		time.Sunday:    "воскресенье",
	}
	return weekdays[weekday]
}
