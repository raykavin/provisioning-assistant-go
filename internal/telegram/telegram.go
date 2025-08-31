package telegram

import (
	"context"
	"fmt"
	"provisioning-assistant/internal/domain"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/gookit/event"
)

type Telegram struct {
	bot          *bot.Bot
	eventManager *event.Manager
	logger       domain.Logger
}

// NewTelegram creates a new Telegram bot adapter with event integration
func NewTelegram(token string, logger domain.Logger, eventManager *event.Manager) (*Telegram, error) {
	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, bot *bot.Bot, update *models.Update) {
			logger.Warnf("Update não tratado: %+v", update)
		}),
	}

	b, err := bot.New(token, opts...)
	if err != nil {
		return nil, err
	}

	adapter := &Telegram{
		bot:          b,
		logger:       logger,
		eventManager: eventManager,
	}

	adapter.registerHandlers()
	adapter.registerEventListeners()

	return adapter, nil
}

// Start begins the Telegram bot polling process
func (t *Telegram) Start(ctx context.Context) {
	t.bot.Start(ctx)
}

// registerHandlers registers bot handlers for messages and callbacks
func (t *Telegram) registerHandlers() {
	t.bot.RegisterHandler(bot.HandlerTypeMessageText, "", bot.MatchTypePrefix, t.handleMessage)
	t.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "", bot.MatchTypePrefix, t.handleCallback)
}

// handleMessage processes incoming text messages from users
func (t *Telegram) handleMessage(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	text := update.Message.Text
	t.logger.Infof("Mensagem recebida do usuário %d: %s", userID, text)

	msgEvent := &domain.MessageEvent{
		UserID:  userID,
		ChatID:  chatID,
		Message: text,
	}

	t.eventManager.MustFire("telegram.message.received", event.M{
		"event": msgEvent,
	})
}

// handleCallback processes incoming callback queries from inline keyboards
func (t *Telegram) handleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	userID := update.CallbackQuery.From.ID
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	data := update.CallbackQuery.Data

	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	t.logger.Infof("Callback recebido do usuário %d: %s", userID, data)

	callbackEvent := &domain.CallbackEvent{
		UserID: userID,
		ChatID: chatID,
		Data:   data,
	}

	t.eventManager.MustFire("telegram.callback.received", event.M{
		"event": callbackEvent,
	})
}

// registerEventListeners registers event listeners for outgoing messages and actions
func (t *Telegram) registerEventListeners() {
	t.eventManager.On("telegram.send.message", event.ListenerFunc(func(e event.Event) error {
		data, ok := e.Get("response").(*domain.MessageResponse)
		if !ok {
			return fmt.Errorf("tipo de resposta de mensagem inválido")
		}

		params := &bot.SendMessageParams{
			ChatID: data.ChatID,
			Text:   data.Text,
		}

		if data.Keyboard != nil {
			params.ReplyMarkup = t.buildKeyboard(data.Keyboard)
		}

		_, err := t.bot.SendMessage(context.Background(), params)
		if err != nil {
			t.logger.Errorf("Erro ao enviar mensagem: %v", err)
			return err
		}

		return nil
	}))

	t.eventManager.On("telegram.send.typing", event.ListenerFunc(func(e event.Event) error {
		chatID, ok := e.Get("chatID").(int64)
		if !ok {
			return fmt.Errorf("tipo de chatID inválido")
		}

		_, err := t.bot.SendChatAction(context.Background(), &bot.SendChatActionParams{
			ChatID: chatID,
			Action: models.ChatActionTyping,
		})

		if err != nil {
			t.logger.Errorf("Erro ao enviar ação de digitação: %v", err)
			return err
		}

		return nil
	}))
}

// buildKeyboard converts domain keyboard to Telegram keyboard markup
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
