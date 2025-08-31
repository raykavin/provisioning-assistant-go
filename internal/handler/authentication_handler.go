package handler

import (
	"fmt"
	"provisioning-assistant/internal/domain"
	"provisioning-assistant/internal/services"
	"strings"
	"time"
)

type AuthenticationHandler struct {
	userService    *services.UserService
	sessionService *services.SessionService
	messenger      *Messenger
	logger         domain.Logger
}

// NewAuthenticationHandler creates a new authentication handler instance
func NewAuthenticationHandler(
	userService *services.UserService,
	sessionService *services.SessionService,
	messenger *Messenger,
	logger domain.Logger,
) *AuthenticationHandler {
	return &AuthenticationHandler{
		userService:    userService,
		sessionService: sessionService,
		messenger:      messenger,
		logger:         logger,
	}
}

// HandleCPFInput processes CPF input for user authentication
func (h *AuthenticationHandler) HandleCPFInput(session *domain.Session, msg *domain.MessageEvent) error {
	cpf := h.sanitizeCPF(msg.Message)

	if !h.isValidCPFFormat(cpf) {
		return h.messenger.SendMessage(msg.ChatID, MSG_CPF_INVALID)
	}

	h.messenger.SendTypingIndicator(msg.ChatID)

	time.Sleep(TIMEOUT_CPF_VALIDATION)

	if err := h.authenticateUser(session, cpf); err != nil {
		h.logger.WithError(err).WithField("cpf", cpf).Debug("Falha na autenticação do CPF")
		session.State = domain.StateWaitingCPF
		h.sessionService.UpdateSession(session)
		return h.messenger.SendMessage(msg.ChatID, MSG_CPF_UNAUTHORIZED)
	}

	return h.sendMainMenu(session)
}

// authenticateUser validates CPF and updates session with user information
func (h *AuthenticationHandler) authenticateUser(session *domain.Session, taxID string) error {
	user := h.userService.ValidateTaxID(taxID)
	if user == nil {
		return fmt.Errorf("usuário com tax id %s não autorizado", taxID)
	}

	session.UserTaxID = taxID
	session.UserName = user.Name
	session.State = domain.StateMainMenu
	h.sessionService.UpdateSession(session)

	h.logger.
		WithField("tax_id", taxID).
		WithField("username", user.Name).
		WithField("chat_id", session.ChatID).
		Info("Usuário autenticado com sucesso")

	return nil
}

// sendMainMenu sends the main menu after successful authentication
func (h *AuthenticationHandler) sendMainMenu(session *domain.Session) error {
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

// sanitizeCPF removes formatting characters from CPF string
func (h *AuthenticationHandler) sanitizeCPF(cpf string) string {
	cpf = strings.ReplaceAll(cpf, ".", "")
	cpf = strings.ReplaceAll(cpf, "-", "")
	cpf = strings.TrimSpace(cpf)
	return cpf
}

// isValidCPFFormat checks if CPF has exactly 11 digits
func (h *AuthenticationHandler) isValidCPFFormat(cpf string) bool {
	return len(cpf) == 11
}

// Logout clears the user session and returns to idle state
func (h *AuthenticationHandler) Logout(session *domain.Session) error {
	session.State = domain.StateIdle
	session.UserTaxID = ""
	session.UserName = ""
	h.sessionService.UpdateSession(session)

	h.logger.WithField("chat_id", session.ChatID).Info("Usuário desconectado")

	return h.messenger.SendMessage(session.ChatID, MSG_EXIT_MESSAGE)
}
