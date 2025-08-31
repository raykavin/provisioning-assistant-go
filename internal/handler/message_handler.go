package handler

import (
	"fmt"
	"provisioning-assistant/internal/domain"
	"provisioning-assistant/internal/services"
	"strings"

	"github.com/gookit/event"
)

type MessageHandler struct {
	eventManager        *event.Manager
	provisioningService *services.ProvisioningService
	userService         *services.UserService
	sessionService      *services.SessionService
	erpService          *services.ErpService
	logger              domain.Logger

	authHandler         *AuthenticationHandler
	provisioningHandler *ProvisioningHandler
	menuHandler         *MenuHandler
	messenger           *Messenger
}

// NewMessageHandler creates a new message handler instance with sub-handlers
func NewMessageHandler(
	eventManager *event.Manager,
	provisioningService *services.ProvisioningService,
	userService *services.UserService,
	sessionService *services.SessionService,
	erpService *services.ErpService,
	logger domain.Logger,
) *MessageHandler {
	messenger := NewMessenger(eventManager)

	return &MessageHandler{
		eventManager:        eventManager,
		provisioningService: provisioningService,
		userService:         userService,
		sessionService:      sessionService,
		erpService:          erpService,
		logger:              logger,
		authHandler:         NewAuthenticationHandler(userService, sessionService, messenger, logger),
		provisioningHandler: NewProvisioningHandler(provisioningService, erpService, sessionService, messenger, eventManager, logger),
		menuHandler:         NewMenuHandler(sessionService, messenger),
		messenger:           messenger,
	}
}

// RegisterEventListeners registers event listeners for messages and callbacks
func (h *MessageHandler) RegisterEventListeners() {
	h.eventManager.On("telegram.message.received", event.ListenerFunc(func(e event.Event) error {
		msgEvent, ok := e.Get("event").(*domain.MessageEvent)
		if !ok {
			return fmt.Errorf("tipo de evento de mensagem inválido")
		}
		return h.handleMessage(msgEvent)
	}))

	h.eventManager.On("telegram.callback.received", event.ListenerFunc(func(e event.Event) error {
		callbackEvent, ok := e.Get("event").(*domain.CallbackEvent)
		if !ok {
			return fmt.Errorf("tipo de evento de callback inválido")
		}
		return h.handleCallback(callbackEvent)
	}))
}

// handleMessage routes messages based on current session state
func (h *MessageHandler) handleMessage(msg *domain.MessageEvent) error {
	session := h.getOrCreateSession(msg.UserID, msg.ChatID)

	switch session.State {
	case domain.StateIdle:
		return h.handleStart(session, msg)
	case domain.StateWaitingCPF:
		return h.authHandler.HandleCPFInput(session, msg)
	case domain.StateWaitingProtocol:
		return h.provisioningHandler.HandleProtocolInput(session, msg)
	default:
		return h.handleStart(session, msg)
	}
}

// handleCallback routes callback queries based on action type
func (h *MessageHandler) handleCallback(callback *domain.CallbackEvent) error {
	session := h.sessionService.GetSession(callback.UserID)
	if session == nil {
		_ = h.sessionService.CreateSession(callback.UserID, callback.ChatID)
		return h.messenger.SendMessage(callback.ChatID, MSG_SESSION_EXPIRED)
	}

	parts := strings.Split(callback.Data, ":")
	if len(parts) == 0 {
		return nil
	}

	action := parts[0]

	switch action {
	case "main_menu":
		return h.menuHandler.HandleMainMenuOption(session, parts[1])
	case "confirm":
		return h.provisioningHandler.HandleConfirmation(session, parts[1])
	default:
		return nil
	}
}

// handleStart initiates the conversation flow and sets waiting for CPF state
func (h *MessageHandler) handleStart(session *domain.Session, msg *domain.MessageEvent) error {
	session.State = domain.StateWaitingCPF
	h.sessionService.UpdateSession(session)

	return h.messenger.SendMessage(msg.ChatID, MSG_WELCOME)
}

// getOrCreateSession retrieves existing session or creates a new one if needed
func (h *MessageHandler) getOrCreateSession(userID, chatID int64) *domain.Session {
	session := h.sessionService.GetSession(userID)
	if session == nil {
		session = h.sessionService.CreateSession(userID, chatID)
	}
	return session
}
