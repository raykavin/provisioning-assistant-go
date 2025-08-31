package handler

import (
	"provisioning-assistant/internal/domain"

	"github.com/gookit/event"
)

// Messenger handles sending messages to users
type Messenger struct {
	eventManager *event.Manager
}

// NewMessenger creates a new messenger instance
func NewMessenger(eventManager *event.Manager) *Messenger {
	return &Messenger{
		eventManager: eventManager,
	}
}

// SendMessage sends a text message to a chat
func (m *Messenger) SendMessage(chatID int64, text string) error {
	response := &domain.MessageResponse{
		ChatID: chatID,
		Text:   text,
	}

	m.eventManager.MustFire("telegram.send.message", event.M{
		"response": response,
	})

	return nil
}

// SendMessageWithKeyboard sends a message with an inline keyboard
func (m *Messenger) SendMessageWithKeyboard(chatID int64, text string, keyboard *domain.Keyboard) error {
	response := &domain.MessageResponse{
		ChatID:   chatID,
		Text:     text,
		Keyboard: keyboard,
	}

	m.eventManager.MustFire("telegram.send.message", event.M{
		"response": response,
	})

	return nil
}

// SendTypingIndicator sends a typing action to show bot is processing
func (m *Messenger) SendTypingIndicator(chatID int64) {
	m.eventManager.MustFire("telegram.send.typing", event.M{
		"chatID": chatID,
	})
}

// SendDocument sends a document/file to a chat
// func (m *Messenger) SendDocument(chatID int64, document *domain.Document) error {
// 	m.eventManager.MustFire("telegram.send.document", event.M{
// 		"chatID":   chatID,
// 		"document": document,
// 	})

// 	return nil
// }

// EditMessage edits an existing message
// func (m *Messenger) EditMessage(chatID int64, messageID int, text string, keyboard *domain.Keyboard) error {
// 	response := &domain.EditMessageResponse{
// 		ChatID:    chatID,
// 		MessageID: messageID,
// 		Text:      text,
// 		Keyboard:  keyboard,
// 	}

// 	m.eventManager.MustFire("telegram.edit.message", event.M{
// 		"response": response,
// 	})

// 	return nil
// }

// DeleteMessage deletes a message
func (m *Messenger) DeleteMessage(chatID int64, messageID int) error {
	m.eventManager.MustFire("telegram.delete.message", event.M{
		"chatID":    chatID,
		"messageID": messageID,
	})

	return nil
}

// AnswerCallbackQuery sends a response to a callback query
func (m *Messenger) AnswerCallbackQuery(callbackID string, text string, showAlert bool) error {
	m.eventManager.MustFire("telegram.answer.callback", event.M{
		"callbackID": callbackID,
		"text":       text,
		"showAlert":  showAlert,
	})

	return nil
}
