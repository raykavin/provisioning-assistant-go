package telegram

import (
	"context"
	"fmt"
	"log"
	"provisioning-assistant/internal/domain"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/gookit/event"
)

type Telegram struct {
	bot          *bot.Bot
	eventManager *event.Manager
}

func NewTelegram(token string, eventManager *event.Manager) (*Telegram, error) {
	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
	}

	b, err := bot.New(token, opts...)
	if err != nil {
		return nil, err
	}

	adapter := &Telegram{
		bot:          b,
		eventManager: eventManager,
	}

	// Register bot handlers
	adapter.registerHandlers()

	// Register event listeners for responses
	adapter.registerEventListeners()

	return adapter, nil
}

func (t *Telegram) Start(ctx context.Context) {
	t.bot.Start(ctx)
}

func (t *Telegram) registerHandlers() {
	t.bot.RegisterHandler(bot.HandlerTypeMessageText, "", bot.MatchTypePrefix, t.handleMessage)
	t.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "", bot.MatchTypePrefix, t.handleCallback)
}

func (t *Telegram) handleMessage(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	log.Printf("Received message from user %d: %s", userID, text)

	// Create message event
	msgEvent := &domain.MessageEvent{
		UserID:  userID,
		ChatID:  chatID,
		Message: text,
	}

	// Emit event to core
	t.eventManager.MustFire("telegram.message.received", event.M{
		"event": msgEvent,
	})
}

func (t *Telegram) handleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	userID := update.CallbackQuery.From.ID
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	data := update.CallbackQuery.Data

	// Answer callback to remove loading state
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	log.Printf("Received callback from user %d: %s", userID, data)

	// Create callback event
	callbackEvent := &domain.CallbackEvent{
		UserID: userID,
		ChatID: chatID,
		Data:   data,
	}

	// Emit event to core
	t.eventManager.MustFire("telegram.callback.received", event.M{
		"event": callbackEvent,
	})
}

func (t *Telegram) registerEventListeners() {
	// Listen for send message events from core
	t.eventManager.On("telegram.send.message", event.ListenerFunc(func(e event.Event) error {
		data, ok := e.Get("response").(*domain.MessageResponse)
		if !ok {
			return fmt.Errorf("invalid message response type")
		}

		params := &bot.SendMessageParams{
			ChatID: data.ChatID,
			Text:   data.Text,
		}

		// Add keyboard if provided
		if data.Keyboard != nil {
			params.ReplyMarkup = t.buildKeyboard(data.Keyboard)
		}

		_, err := t.bot.SendMessage(context.Background(), params)
		if err != nil {
			log.Printf("Error sending message: %v", err)
			return err
		}

		return nil
	}))

	// Listen for typing action events
	t.eventManager.On("telegram.send.typing", event.ListenerFunc(func(e event.Event) error {
		chatID, ok := e.Get("chatID").(int64)
		if !ok {
			return fmt.Errorf("invalid chatID type")
		}

		_, err := t.bot.SendChatAction(context.Background(), &bot.SendChatActionParams{
			ChatID: chatID,
			Action: models.ChatActionTyping,
		})

		if err != nil {
			log.Printf("Error sending typing action: %v", err)
			return err
		}

		return nil
	}))
}

func (t *Telegram) buildKeyboard(keyboard *domain.Keyboard) models.ReplyMarkup {
	if keyboard.Inline {
		var rows [][]models.InlineKeyboardButton
		for _, row := range keyboard.Buttons {
			var buttons []models.InlineKeyboardButton
			for _, btn := range row {
				buttons = append(buttons, models.InlineKeyboardButton{
					Text:         btn.Text,
					CallbackData: btn.Data,
				})
			}
			rows = append(rows, buttons)
		}
		return &models.InlineKeyboardMarkup{
			InlineKeyboard: rows,
		}
	}

	// Reply keyboard
	var rows [][]models.KeyboardButton
	for _, row := range keyboard.Buttons {
		var buttons []models.KeyboardButton
		for _, btn := range row {
			buttons = append(buttons, models.KeyboardButton{
				Text: btn.Text,
			})
		}
		rows = append(rows, buttons)
	}
	return &models.ReplyKeyboardMarkup{
		Keyboard:       rows,
		ResizeKeyboard: true,
	}
}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	// Default handler for unhandled updates
	log.Printf("Unhandled update: %+v", update)
}
