package bot

import (
	"log"
	"volvomaster/internal/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api       *tgbotapi.BotAPI
	dbService *services.DatabaseService
	updates   tgbotapi.UpdatesChannel
	handlers  *Handlers
}

func NewBot(token string, dbService *services.DatabaseService) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	api.Debug = true

	handlers := NewHandlers(api, dbService)

	return &Bot{
		api:       api,
		dbService: dbService,
		handlers:  handlers,
	}, nil
}

func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)
	b.updates = updates

	for update := range updates {
		if update.Message != nil {
			go func(msg *tgbotapi.Message) {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Паника в обработке сообщения: %v", r)
					}
				}()
				b.handlers.HandleMessage(msg)
			}(update.Message)
		} else if update.CallbackQuery != nil {
			go func(callback *tgbotapi.CallbackQuery) {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Паника в обработке callback: %v", r)
					}
				}()
				b.handlers.HandleCallbackQuery(callback)
			}(update.CallbackQuery)
		} else if update.MyChatMember != nil {
			go func(update *tgbotapi.ChatMemberUpdated) {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Паника в обработке chat member update: %v", r)
					}
				}()
				b.handlers.HandleChatMemberUpdate(update)
			}(update.MyChatMember)
		}
	}
}

func (b *Bot) Stop() {
	b.api.StopReceivingUpdates()
} 