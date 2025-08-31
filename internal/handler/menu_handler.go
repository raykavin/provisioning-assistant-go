package handler

import (
	"fmt"
	"provisioning-assistant/internal/domain"
	"provisioning-assistant/internal/services"
)

type MenuHandler struct {
	sessionService *services.SessionService
	messenger      *Messenger
}

// NewMenuHandler creates a new menu handler instance
func NewMenuHandler(
	sessionService *services.SessionService,
	messenger *Messenger,
) *MenuHandler {
	return &MenuHandler{
		sessionService: sessionService,
		messenger:      messenger,
	}
}

// HandleMainMenuOption processes main menu selection and routes to appropriate handler
func (h *MenuHandler) HandleMainMenuOption(session *domain.Session, option string) error {
	switch option {
	case "provision":
		return h.handleProvisionOption(session)
	case "exit":
		return h.handleExitOption(session)
	default:
		return h.sendMainMenu(session)
	}
}

// handleProvisionOption handles equipment provisioning menu selection
func (h *MenuHandler) handleProvisionOption(session *domain.Session) error {
	session.State = domain.StateWaitingProtocol
	h.sessionService.UpdateSession(session)
	return h.messenger.SendMessage(session.ChatID, MSG_REQUEST_PROTOCOL)
}

// handleExitOption handles exit menu selection and resets session
func (h *MenuHandler) handleExitOption(session *domain.Session) error {
	session.State = domain.StateIdle
	h.sessionService.UpdateSession(session)
	return h.messenger.SendMessage(session.ChatID, MSG_EXIT_MESSAGE)
}

// sendMainMenu sends the main menu with inline keyboard buttons
func (h *MenuHandler) sendMainMenu(session *domain.Session) error {
	keyboard := &domain.Keyboard{
		Inline: true,
		Buttons: [][]domain.Button{
			{{Text: MSG_MENU_PROVISION, Data: "main_menu:provision"}},
			{{Text: MSG_MENU_EXIT, Data: "main_menu:exit"}},
		},
	}

	message := fmt.Sprintf(MSG_USER_GREETING, session.UserName)
	return h.messenger.SendMessageWithKeyboard(session.ChatID, message, keyboard)
}

// SendContextualMenu sends appropriate menu based on current session state
func (h *MenuHandler) SendContextualMenu(session *domain.Session) error {
	switch session.State {
	case domain.StateMainMenu:
		return h.sendMainMenu(session)
	case domain.StateWaitingProtocol:
		return h.messenger.SendMessage(session.ChatID, MSG_REQUEST_PROTOCOL)
	case domain.StateWaitingCPF:
		return h.messenger.SendMessage(session.ChatID, MSG_WELCOME)
	default:
		return h.sendMainMenu(session)
	}
}