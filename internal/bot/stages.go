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

type StageHandlers struct {
	api       *tgbotapi.BotAPI
	dbService *services.DatabaseService
}

func NewStageHandlers(api *tgbotapi.BotAPI, dbService *services.DatabaseService) *StageHandlers {
	return &StageHandlers{
		api:       api,
		dbService: dbService,
	}
}

func (s *StageHandlers) StartNewRequest(session *models.UserSession) {
	ctx := context.Background()
	chatID := session.ChatID

	// Создаем новую заявку
	request := &models.ServiceRequest{
		UserID:    session.UserID,
		ChatID:    chatID,
		Stage:     models.StagePersonalInfo,
		Status:    "in_progress",
		CreatedAt: time.Now(),
	}

	err := s.dbService.SaveServiceRequest(ctx, request)
	if err != nil {
		log.Printf("Ошибка сохранения заявки: %v", err)
		s.sendMessage(chatID, "Произошла ошибка при создании заявки.")
		return
	}

	// Обновляем сессию
	session.Stage = models.StagePersonalInfo
	session.RequestID = request.ID
	session.Data = make(map[string]interface{})
	session.Data["step"] = "name"

	err = s.dbService.SaveUserSession(ctx, session)
	if err != nil {
		log.Printf("Ошибка сохранения сессии: %v", err)
	}

	welcomeText := `
🛠️ Добро пожаловать в сервис записи на обслуживание Volvo!

Я помогу вам записаться на диагностику или ремонт вашего автомобиля.

📝 ЭТАП 1: Контактная информация

Как к вам обращаться? Введите ваше имя:`

	s.sendMessage(chatID, welcomeText)
}

func (s *StageHandlers) HandlePersonalInfo(message *tgbotapi.Message, session *models.UserSession) {
	ctx := context.Background()
	chatID := message.Chat.ID
	text := strings.TrimSpace(message.Text)

	if text == "" {
		s.sendMessage(chatID, "Пожалуйста, введите корректную информацию.")
		return
	}

	step, ok := session.Data["step"].(string)
	if !ok {
		step = "name"
	}

	request, err := s.dbService.GetServiceRequest(ctx, session.RequestID)
	if err != nil {
		log.Printf("Ошибка получения заявки: %v", err)
		s.sendMessage(chatID, "Произошла ошибка. Начните заново с /start")
		return
	}

	switch step {
	case "name":
		request.Name = text
		session.Data["step"] = "contact"

		if err := s.dbService.SaveServiceRequest(ctx, request); err != nil {
			log.Printf("Ошибка сохранения заявки: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}
		if err := s.dbService.SaveUserSession(ctx, session); err != nil {
			log.Printf("Ошибка сохранения сессии: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}

		s.sendMessage(chatID, "Отлично! Теперь укажите ваш номер телефона или Telegram для связи:")

	case "contact":
		request.Contact = text
		session.Stage = models.StageCarInfo
		session.Data["step"] = "model"

		if err := s.dbService.SaveServiceRequest(ctx, request); err != nil {
			log.Printf("Ошибка сохранения заявки: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}
		if err := s.dbService.SaveUserSession(ctx, session); err != nil {
			log.Printf("Ошибка сохранения сессии: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}

		carInfoText := `
✅ Контактная информация сохранена!

🚗 ЭТАП 2: Информация об автомобиле

Какая у вас модель Volvo? (например: XC90, XC60, S60, V90 и т.д.)`

		s.sendMessage(chatID, carInfoText)
	}
}

func (s *StageHandlers) HandleCarInfo(message *tgbotapi.Message, session *models.UserSession) {
	ctx := context.Background()
	chatID := message.Chat.ID
	text := strings.TrimSpace(message.Text)

	if text == "" {
		s.sendMessage(chatID, "Пожалуйста, введите корректную информацию.")
		return
	}

	step, ok := session.Data["step"].(string)
	if !ok {
		step = "model"
	}

	request, err := s.dbService.GetServiceRequest(ctx, session.RequestID)
	if err != nil {
		log.Printf("Ошибка получения заявки: %v", err)
		s.sendMessage(chatID, "Произошла ошибка. Начните заново с /start")
		return
	}

	switch step {
	case "model":
		request.VolvoModel = text
		session.Data["step"] = "year"

		if err := s.dbService.SaveServiceRequest(ctx, request); err != nil {
			log.Printf("Ошибка сохранения заявки: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}
		if err := s.dbService.SaveUserSession(ctx, session); err != nil {
			log.Printf("Ошибка сохранения сессии: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}

		s.sendMessage(chatID, "Год выпуска автомобиля:")

	case "year":
		request.Year = text
		session.Data["step"] = "engine_type"

		if err := s.dbService.SaveServiceRequest(ctx, request); err != nil {
			log.Printf("Ошибка сохранения заявки: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}
		if err := s.dbService.SaveUserSession(ctx, session); err != nil {
			log.Printf("Ошибка сохранения сессии: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}

		// Создаем клавиатуру для выбора типа двигателя
		keyboard := tgbotapi.NewInlineKeyboardMarkup()
		for _, engineType := range models.EngineTypes {
			row := tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(engineType, "engine_"+engineType),
			)
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
		}

		msg := tgbotapi.NewMessage(chatID, "Выберите тип двигателя:")
		msg.ReplyMarkup = keyboard
		s.api.Send(msg)

	case "engine_volume":
		request.EngineVolume = text
		session.Data["step"] = "mileage"

		if err := s.dbService.SaveServiceRequest(ctx, request); err != nil {
			log.Printf("Ошибка сохранения заявки: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}
		if err := s.dbService.SaveUserSession(ctx, session); err != nil {
			log.Printf("Ошибка сохранения сессии: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}

		s.sendMessage(chatID, "Пробег автомобиля на текущий момент (в км):")

	case "mileage":
		request.Mileage = text
		session.Stage = models.StageProblemInfo
		session.Data["step"] = "problem"

		if err := s.dbService.SaveServiceRequest(ctx, request); err != nil {
			log.Printf("Ошибка сохранения заявки: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}
		if err := s.dbService.SaveUserSession(ctx, session); err != nil {
			log.Printf("Ошибка сохранения сессии: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}

		problemInfoText := `
✅ Информация об автомобиле сохранена!

🔧 ЭТАП 3: Описание проблемы

Что именно вас беспокоит или что нужно сделать?

Например: "гремит спереди", "нужно заменить масло", "ошибка по двигателю", "не работает климат"`

		s.sendMessage(chatID, problemInfoText)
	}
}

func (s *StageHandlers) HandleProblemInfo(message *tgbotapi.Message, session *models.UserSession) {
	ctx := context.Background()
	chatID := message.Chat.ID
	text := strings.TrimSpace(message.Text)

	if text == "" {
		s.sendMessage(chatID, "Пожалуйста, введите корректную информацию.")
		return
	}

	step, ok := session.Data["step"].(string)
	if !ok {
		step = "problem"
	}

	request, err := s.dbService.GetServiceRequest(ctx, session.RequestID)
	if err != nil {
		log.Printf("Ошибка получения заявки: %v", err)
		s.sendMessage(chatID, "Произошла ошибка. Начните заново с /start")
		return
	}

	switch step {
	case "problem":
		request.Problem = text
		session.Data["step"] = "problem_appeared"

		if err := s.dbService.SaveServiceRequest(ctx, request); err != nil {
			log.Printf("Ошибка сохранения заявки: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}
		if err := s.dbService.SaveUserSession(ctx, session); err != nil {
			log.Printf("Ошибка сохранения сессии: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}

		// Создаем клавиатуру для выбора когда появилась проблема
		keyboard := tgbotapi.NewInlineKeyboardMarkup()
		for _, option := range models.ProblemAppeared {
			row := tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(option, "appeared_"+option),
			)
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
		}

		msg := tgbotapi.NewMessage(chatID, "Когда впервые появилась проблема?")
		msg.ReplyMarkup = keyboard
		s.api.Send(msg)

	case "problem_frequency":
		request.ProblemFrequency = text
		session.Data["step"] = "safety_impact"

		if err := s.dbService.SaveServiceRequest(ctx, request); err != nil {
			log.Printf("Ошибка сохранения заявки: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}
		if err := s.dbService.SaveUserSession(ctx, session); err != nil {
			log.Printf("Ошибка сохранения сессии: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}

		s.sendMessage(chatID, `Влияет ли это на движение или безопасность?

Например: "машина не заводится", "перестали работать тормоза", "не влияет на безопасность"`)

	case "safety_impact":
		request.SafetyImpact = text
		session.Data["step"] = "previous_repairs"

		if err := s.dbService.SaveServiceRequest(ctx, request); err != nil {
			log.Printf("Ошибка сохранения заявки: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}
		if err := s.dbService.SaveUserSession(ctx, session); err != nil {
			log.Printf("Ошибка сохранения сессии: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}

		s.sendMessage(chatID, `Уже предпринимались попытки ремонта или диагностики?

Если да — что делали и где? Если нет, напишите "нет"`)

	case "previous_repairs":
		request.PreviousRepairs = text
		session.Data["step"] = "recent_changes"

		if err := s.dbService.SaveServiceRequest(ctx, request); err != nil {
			log.Printf("Ошибка сохранения заявки: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}
		if err := s.dbService.SaveUserSession(ctx, session); err != nil {
			log.Printf("Ошибка сохранения сессии: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}

		s.sendMessage(chatID, `Меняли ли что-то недавно?

Например: "меняли подвеску месяц назад", "недавно меняли масло". Если ничего не меняли, напишите "нет"`)

	case "recent_changes":
		request.RecentChanges = text
		session.Stage = models.StageDateSelection

		if err := s.dbService.SaveServiceRequest(ctx, request); err != nil {
			log.Printf("Ошибка сохранения заявки: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}
		if err := s.dbService.SaveUserSession(ctx, session); err != nil {
			log.Printf("Ошибка сохранения сессии: %v", err)
			s.sendMessage(chatID, "Произошла ошибка при сохранении. Попробуйте еще раз.")
			return
		}

		// Переходим к выбору даты
		s.showAvailableDates(chatID)
	}
}

func (s *StageHandlers) showAvailableDates(chatID int64) {
	// Используем fallback - генерируем даты вручную
	s.showFallbackDates(chatID)
}

// showFallbackDates показывает даты, сгенерированные вручную, если календарь недоступен
func (s *StageHandlers) showFallbackDates(chatID int64) {
	problemInfoComplete := `
✅ Информация о проблеме сохранена!

📅 ЭТАП 4: Выбор даты и времени

Выберите удобную дату и время для визита:`

	s.sendMessage(chatID, problemInfoComplete)

	// Генерируем даты на следующие 2 недели (рабочие дни, 9:00-17:00)
	keyboard := tgbotapi.NewInlineKeyboardMarkup()
	now := time.Now()

	// Добавляем даты на следующие 10 рабочих дней
	dateCount := 0
	for days := 1; days <= 14 && dateCount < 10; days++ {
		date := now.AddDate(0, 0, days)

		// Пропускаем выходные
		if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
			continue
		}

		// Добавляем временные слоты (9:00, 11:00, 13:00, 15:00)
		for hour := 9; hour <= 15; hour += 2 {
			slot := time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, date.Location())
			dateStr := slot.Format("02.01.2006 15:04")
			weekday := getWeekdayRussian(slot.Weekday())
			buttonText := fmt.Sprintf("%s (%s)", dateStr, weekday)

			row := tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(buttonText, "date_"+strconv.FormatInt(slot.Unix(), 10)),
			)
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
			dateCount++

			if dateCount >= 10 {
				break
			}
		}
	}

	msg := tgbotapi.NewMessage(chatID, "Доступные даты и время (автоматически сгенерированные):")
	msg.ReplyMarkup = keyboard
	s.api.Send(msg)
}

func (s *StageHandlers) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	_, err := s.api.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения пользователю %d: %v", chatID, err)
	}
}
