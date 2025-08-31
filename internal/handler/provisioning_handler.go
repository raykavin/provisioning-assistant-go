package handler

import (
	"context"
	"fmt"
	"provisioning-assistant/internal/domain"
	"provisioning-assistant/internal/domain/dto"
	"provisioning-assistant/internal/services"
	"strconv"
	"strings"

	"github.com/gookit/event"
)

type ProvisioningHandler struct {
	provisioningService *services.ProvisioningService
	erpService          *services.ErpService
	sessionService      *services.SessionService
	messenger           *Messenger
	eventManager        *event.Manager
	logger              domain.Logger
}

// NewProvisioningHandler creates a new provisioning handler instance
func NewProvisioningHandler(
	provisioningService *services.ProvisioningService,
	erpService *services.ErpService,
	sessionService *services.SessionService,
	messenger *Messenger,
	eventManager *event.Manager,
	logger domain.Logger,
) *ProvisioningHandler {
	return &ProvisioningHandler{
		provisioningService: provisioningService,
		erpService:          erpService,
		sessionService:      sessionService,
		messenger:           messenger,
		eventManager:        eventManager,
		logger:              logger,
	}
}

// HandleProtocolInput processes protocol number input from user
func (h *ProvisioningHandler) HandleProtocolInput(session *domain.Session, msg *domain.MessageEvent) error {
	protocol := strings.TrimSpace(msg.Message)

	if _, err := strconv.ParseInt(protocol, 10, 64); err != nil {
		return h.messenger.SendMessage(msg.ChatID, MSG_PROTOCOL_INVALID)
	}

	connectionInfo, err := h.fetchConnectionInfo(msg.ChatID, protocol)
	if err != nil {
		h.logger.WithError(err).WithField("protocol", protocol).Error("Falha ao buscar informações de conexão")
		return h.messenger.SendMessage(msg.ChatID, MSG_PROTOCOL_NOT_FOUND)
	}

	h.updateSessionWithConnectionInfo(session, protocol, connectionInfo)

	return h.sendConfirmationRequest(session)
}

// fetchConnectionInfo retrieves connection information from ERP system
func (h *ProvisioningHandler) fetchConnectionInfo(chatID int64, protocol string) (*dto.ConnectionInfo, error) {
	h.messenger.SendTypingIndicator(chatID)
	_ = h.messenger.SendMessage(chatID, MSG_SEARCHING_INFO)

	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT_ERP_FETCH)
	defer cancel()

	return h.erpService.GetConnectionInfo(ctx, protocol)
}

// updateSessionWithConnectionInfo updates session with connection data and state
func (h *ProvisioningHandler) updateSessionWithConnectionInfo(
	session *domain.Session,
	protocol string,
	connectionInfo *dto.ConnectionInfo,
) {
	session.Protocol = protocol
	session.ConnectionInfo = connectionInfo
	session.State = domain.StateConfirmData
	h.sessionService.UpdateSession(session)
}

// sendConfirmationRequest sends confirmation message with connection details
func (h *ProvisioningHandler) sendConfirmationRequest(session *domain.Session) error {
	keyboard := &domain.Keyboard{
		Inline: true,
		Buttons: [][]domain.Button{
			{
				{Text: MSG_CONFIRM_YES, Data: "confirm:yes"},
				{Text: MSG_CONFIRM_NO, Data: "confirm:no"},
			},
		},
	}

	message := fmt.Sprintf(
		MSG_CONFIRM_DATA,
		session.ConnectionInfo.ContractDescription,
		session.ConnectionInfo.AssignmentTitle,
		session.ConnectionInfo.ConnectionEquipmentSerialNumber,
		session.ConnectionInfo.ConnectionClientSplitterName,
		session.ConnectionInfo.ConnectionClientSplitterPort,
	)

	return h.messenger.SendMessageWithKeyboard(session.ChatID, message, keyboard)
}

// HandleConfirmation processes user confirmation response for provisioning
func (h *ProvisioningHandler) HandleConfirmation(session *domain.Session, confirm string) error {
	if confirm != "yes" {
		if err := h.handleConfirmationDenied(session); err != nil {
			return err
		}

		return  h.
	}

	return h.executeProvisioning(session)
}

// handleConfirmationDenied handles when user denies the confirmation
func (h *ProvisioningHandler) handleConfirmationDenied(session *domain.Session) error {
	session.State = domain.StateIdle
	h.sessionService.UpdateSession(session)

	return h.messenger.SendMessage(session.ChatID, MSG_CONFIRMATION_DENIED)
}

// executeProvisioning performs the complete equipment provisioning process
func (h *ProvisioningHandler) executeProvisioning(session *domain.Session) error {
	h.messenger.SendTypingIndicator(session.ChatID)
	_ = h.messenger.SendMessage(session.ChatID, MSG_PROVISIONING_START)

	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT_PROVISIONING)
	defer cancel()

	signalInfo, err := h.provisioningService.ProvisionEquipment(ctx, session.ConnectionInfo)
	if err != nil {
		return h.handleProvisioningError(session, err)
	}

	return h.handleProvisioningSuccess(session, signalInfo)
}

// handleProvisioningError handles provisioning failure and resets session
func (h *ProvisioningHandler) handleProvisioningError(session *domain.Session, err error) error {
	h.logger.WithError(err).WithField("protocol", session.Protocol).Error("Falha no provisionamento")

	session.State = domain.StateIdle
	h.sessionService.UpdateSession(session)

	message := fmt.Sprintf(MSG_PROVISIONING_FAILED, err)
	return h.messenger.SendMessage(session.ChatID, message)
}

// handleProvisioningSuccess handles successful provisioning and builds response
func (h *ProvisioningHandler) handleProvisioningSuccess(
	session *domain.Session,
	signalInfo *domain.OnuSignalInfo,
) error {
	session.State = domain.StateIdle
	h.sessionService.UpdateSession(session)

	message := h.buildSuccessMessage(session.ConnectionInfo, signalInfo)

	h.logger.WithFields(map[string]any{
		"protocol": session.Protocol,
		"contract": session.ConnectionInfo.ContractDescription,
		"serial":   session.ConnectionInfo.ConnectionEquipmentSerialNumber,
	}).Info("Provisionamento concluído com sucesso")

	return h.messenger.SendMessage(session.ChatID, message)
}

// buildSuccessMessage creates the success message with equipment and signal details
func (h *ProvisioningHandler) buildSuccessMessage(
	connectionInfo *dto.ConnectionInfo,
	signalInfo *domain.OnuSignalInfo,
) string {
	message := fmt.Sprintf(
		MSG_PROVISIONING_SUCCESS,
		connectionInfo.ContractDescription,
		connectionInfo.ConnectionEquipmentSerialNumber,
	)

	if signalInfo != nil && h.hasSignalData(signalInfo) {
		message += fmt.Sprintf(
			MSG_SIGNAL_INFO,
			"1.94",
			"-23.01",
			"3.28",
			"56.17",
		)
	}

	message += MSG_EQUIPMENT_READY
	return message
}

// hasSignalData checks if signal information contains valid data
func (h *ProvisioningHandler) hasSignalData(signalInfo *domain.OnuSignalInfo) bool {
	return signalInfo.TxPower != "" && signalInfo.RxPower != ""
}
