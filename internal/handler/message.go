package handler

import (
	"fmt"
	"provisioning-assistant/internal/domain"
	"provisioning-assistant/internal/services"
	"strings"
	"time"

	"github.com/gookit/event"
)

type MessageHandler struct {
	eventManager        *event.Manager
	provisioningService *services.ProvisioningService
	userService         *services.UserService
	sessionService      *services.SessionService
}

func NewMessageHandler(
	eventManager *event.Manager,
	provisioningService *services.ProvisioningService,
	userService *services.UserService,
	sessionService *services.SessionService,
) *MessageHandler {
	return &MessageHandler{
		eventManager:        eventManager,
		provisioningService: provisioningService,
		userService:         userService,
		sessionService:      sessionService,
	}
}

func (h *MessageHandler) RegisterEventListeners() {
	// Handle text messages
	h.eventManager.On("telegram.message.received", event.ListenerFunc(func(e event.Event) error {
		msgEvent, ok := e.Get("event").(*domain.MessageEvent)
		if !ok {
			return fmt.Errorf("invalid message event type")
		}

		return h.handleMessage(msgEvent)
	}))

	// Handle callback queries
	h.eventManager.On("telegram.callback.received", event.ListenerFunc(func(e event.Event) error {
		callbackEvent, ok := e.Get("event").(*domain.CallbackEvent)
		if !ok {
			return fmt.Errorf("invalid callback event type")
		}

		return h.handleCallback(callbackEvent)
	}))
}

func (h *MessageHandler) handleMessage(msg *domain.MessageEvent) error {
	// Get or create session
	session := h.sessionService.GetSession(msg.UserID)
	if session == nil {
		session = h.sessionService.CreateSession(msg.UserID, msg.ChatID)
	}

	// Handle based on current state
	switch session.State {
	case domain.StateIdle:
		return h.handleStart(session, msg)
	case domain.StateWaitingCPF:
		return h.handleCPF(session, msg)
	case domain.StateWaitingContract:
		return h.handleContract(session, msg)
	case domain.StateWaitingSerial:
		return h.handleSerial(session, msg)
	case domain.StateWaitingOldSerial:
		return h.handleOldSerial(session, msg)
	case domain.StateWaitingOLT:
		return h.handleOLT(session, msg)
	case domain.StateWaitingSlot:
		return h.handleSlot(session, msg)
	case domain.StateWaitingPort:
		return h.handlePort(session, msg)
	default:
		return h.handleStart(session, msg)
	}
}

func (h *MessageHandler) handleCallback(callback *domain.CallbackEvent) error {
	session := h.sessionService.GetSession(callback.UserID)
	if session == nil {
		_ = h.sessionService.CreateSession(callback.UserID, callback.ChatID)
		return h.sendMessage(callback.ChatID, "Sessão expirada. Por favor, digite /start para começar novamente.")
	}

	// Parse callback data
	parts := strings.Split(callback.Data, ":")
	if len(parts) == 0 {
		return nil
	}

	action := parts[0]

	switch action {
	case "main_menu":
		return h.handleMainMenuOption(session, parts[1])
	case "service":
		return h.handleServiceOption(session, parts[1])
	case "maintenance":
		return h.handleMaintenanceOption(session, parts[1])
	case "confirm":
		return h.handleConfirmation(session, parts[1])
	case "olt":
		return h.handleOLTSelection(session, parts[1])
	default:
		return nil
	}
}

func (h *MessageHandler) handleStart(session *domain.Session, msg *domain.MessageEvent) error {
	session.State = domain.StateWaitingCPF
	h.sessionService.UpdateSession(session)

	return h.sendMessage(
		msg.ChatID,
		"🤖 Provisionamento de Equipamentos - Fibralink\n\n"+
			"Para continuar, preciso verificar sua identidade.\n"+
			"Por favor, digite seu CPF (apenas números):",
	)
}

func (h *MessageHandler) handleCPF(session *domain.Session, msg *domain.MessageEvent) error {
	cpf := strings.ReplaceAll(msg.Message, ".", "")
	cpf = strings.ReplaceAll(cpf, "-", "")
	cpf = strings.TrimSpace(cpf)

	if len(cpf) != 11 {
		return h.sendMessage(msg.ChatID, "❌ CPF inválido. Digite apenas os 11 dígitos do CPF.")
	}

	// Send typing action
	h.eventManager.MustFire("telegram.send.typing", event.M{"chatID": msg.ChatID})

	// Simulate validation delay
	time.Sleep(2 * time.Second)

	// Validate CPF
	user := h.userService.ValidateCPF(cpf)
	if user == nil {
		session.State = domain.StateWaitingCPF
		h.sessionService.UpdateSession(session)
		return h.sendMessage(
			msg.ChatID,
			"❌ CPF não encontrado em nossa base de dados.\n"+
				"Por favor, verifique o número e tente novamente:",
		)
	}

	// Update session
	session.CPF = cpf
	session.UserName = user.Name
	session.State = domain.StateMainMenu
	h.sessionService.UpdateSession(session)

	// Send main menu
	return h.sendMainMenu(session)
}

func (h *MessageHandler) sendMainMenu(session *domain.Session) error {
	keyboard := &domain.Keyboard{
		Inline: true,
		Buttons: [][]domain.Button{
			{{Text: "🔧 Provisionar Equipamento", Data: "main_menu:provision"}},
			{{Text: "❌ Sair", Data: "main_menu:exit"}},
		},
	}

	return h.sendMessageWithKeyboard(
		session.ChatID,
		fmt.Sprintf("✅ Olá, %s!\n\nO que você deseja fazer?", session.UserName),
		keyboard,
	)
}

func (h *MessageHandler) handleMainMenuOption(session *domain.Session, option string) error {
	switch option {
	case "provision":
		session.State = domain.StateServiceSelection
		h.sessionService.UpdateSession(session)
		return h.sendServiceMenu(session)
	case "exit":
		session.State = domain.StateIdle
		h.sessionService.UpdateSession(session)
		return h.sendMessage(session.ChatID, "👋 Obrigado por usar nosso sistema. Até logo!")
	default:
		return nil
	}
}

func (h *MessageHandler) sendServiceMenu(session *domain.Session) error {
	keyboard := &domain.Keyboard{
		Inline: true,
		Buttons: [][]domain.Button{
			{{Text: "✅ Ativação", Data: "service:activation"}},
			{{Text: "🔧 Manutenção", Data: "service:maintenance"}},
			{{Text: "📍 Mudança de Endereço", Data: "service:address_change"}},
			{{Text: "❌ Voltar", Data: "main_menu:back"}},
		},
	}

	return h.sendMessageWithKeyboard(
		session.ChatID,
		"📋 Qual tipo de serviço você está realizando?",
		keyboard,
	)
}

func (h *MessageHandler) handleServiceOption(session *domain.Session, option string) error {
	switch option {
	case "activation":
		session.ServiceType = domain.ServiceActivation
		session.State = domain.StateWaitingContract
		h.sessionService.UpdateSession(session)
		return h.sendMessage(session.ChatID, "📄 Por favor, informe o número do contrato do cliente:")

	case "maintenance":
		session.ServiceType = domain.ServiceMaintenance
		session.State = domain.StateMaintenanceMenu
		h.sessionService.UpdateSession(session)
		return h.sendMaintenanceMenu(session)

	case "address_change":
		session.ServiceType = domain.ServiceAddressChange
		session.State = domain.StateWaitingOldSerial
		h.sessionService.UpdateSession(session)
		return h.sendMessage(session.ChatID, "🔍 Por favor, informe o serial da ONU atual:")

	default:
		return h.sendServiceMenu(session)
	}
}

func (h *MessageHandler) sendMaintenanceMenu(session *domain.Session) error {
	keyboard := &domain.Keyboard{
		Inline: true,
		Buttons: [][]domain.Button{
			{{Text: "🔄 Troca de ONU", Data: "maintenance:onu_change"}},
			{{Text: "❌ Voltar", Data: "service:back"}},
		},
	}

	return h.sendMessageWithKeyboard(
		session.ChatID,
		"🔧 Qual tipo de manutenção você deseja realizar?",
		keyboard,
	)
}

func (h *MessageHandler) handleMaintenanceOption(session *domain.Session, option string) error {
	switch option {
	case "onu_change":
		session.MaintenanceType = domain.MaintenanceONUChange
		session.State = domain.StateWaitingOldSerial
		h.sessionService.UpdateSession(session)
		return h.sendMessage(session.ChatID, "🔍 Por favor, informe o serial da ONU atual que será substituída:")

	default:
		return h.sendMaintenanceMenu(session)
	}
}

func (h *MessageHandler) handleContract(session *domain.Session, msg *domain.MessageEvent) error {
	contract := strings.TrimSpace(msg.Message)
	if len(contract) < 5 {
		return h.sendMessage(msg.ChatID, "❌ Contrato inválido. Por favor, digite um contrato válido:")
	}

	session.Contract = contract
	session.State = domain.StateWaitingSerial
	h.sessionService.UpdateSession(session)

	return h.sendMessage(msg.ChatID, "📟 Agora informe o serial do equipamento:")
}

func (h *MessageHandler) handleSerial(session *domain.Session, msg *domain.MessageEvent) error {
	serial := strings.ToUpper(strings.TrimSpace(msg.Message))
	if len(serial) < 5 {
		return h.sendMessage(msg.ChatID, "❌ Serial inválido. Por favor, digite um serial válido:")
	}

	session.SerialNumber = serial
	session.State = domain.StateConfirmData
	h.sessionService.UpdateSession(session)

	// Send confirmation
	keyboard := &domain.Keyboard{
		Inline: true,
		Buttons: [][]domain.Button{
			{
				{Text: "✅ Sim", Data: "confirm:yes"},
				{Text: "❌ Não", Data: "confirm:no"},
			},
		},
	}

	message := fmt.Sprintf(
		"📋 Confirme os dados:\n\n"+
			"📄 Contrato: %s\n"+
			"📟 Serial: %s\n\n"+
			"Os dados estão corretos?",
		session.Contract,
		session.SerialNumber,
	)

	return h.sendMessageWithKeyboard(msg.ChatID, message, keyboard)
}

func (h *MessageHandler) handleOldSerial(session *domain.Session, msg *domain.MessageEvent) error {
	serial := strings.ToUpper(strings.TrimSpace(msg.Message))
	if len(serial) < 5 {
		return h.sendMessage(msg.ChatID, "❌ Serial inválido. Por favor, digite um serial válido:")
	}

	session.OldSerialNumber = serial

	if session.ServiceType == domain.ServiceMaintenance {
		session.State = domain.StateWaitingContract
		h.sessionService.UpdateSession(session)
		return h.sendMessage(msg.ChatID, "📄 Por favor, informe o número do contrato do cliente:")
	} else if session.ServiceType == domain.ServiceAddressChange {
		session.State = domain.StateWaitingOLT
		h.sessionService.UpdateSession(session)
		return h.sendOLTMenu(session)
	}

	return nil
}

func (h *MessageHandler) sendOLTMenu(session *domain.Session) error {
	var buttons [][]domain.Button
	for i, olt := range domain.OLTOptions {
		buttons = append(buttons, []domain.Button{
			{Text: olt, Data: fmt.Sprintf("olt:%d", i)},
		})
	}

	keyboard := &domain.Keyboard{
		Inline:  true,
		Buttons: buttons,
	}

	return h.sendMessageWithKeyboard(
		session.ChatID,
		"🌐 Selecione a OLT de destino:",
		keyboard,
	)
}

func (h *MessageHandler) handleOLTSelection(session *domain.Session, index string) error {
	var idx int
	fmt.Sscanf(index, "%d", &idx)

	if idx >= 0 && idx < len(domain.OLTOptions) {
		session.OLT = domain.OLTOptions[idx]
		session.State = domain.StateWaitingSlot
		h.sessionService.UpdateSession(session)
		return h.sendMessage(session.ChatID, "🔌 Informe o Slot da OLT (ex: 1, 2, 3...):")
	}

	return h.sendOLTMenu(session)
}

func (h *MessageHandler) handleOLT(session *domain.Session, msg *domain.MessageEvent) error {
	// This is handled by callback
	return h.sendOLTMenu(session)
}

func (h *MessageHandler) handleSlot(session *domain.Session, msg *domain.MessageEvent) error {
	slot := strings.TrimSpace(msg.Message)
	if len(slot) == 0 {
		return h.sendMessage(msg.ChatID, "❌ Slot inválido. Por favor, digite um slot válido:")
	}

	session.Slot = slot
	session.State = domain.StateWaitingPort
	h.sessionService.UpdateSession(session)

	return h.sendMessage(msg.ChatID, "🔌 Informe a Porta da OLT (ex: 1, 2, 3...):")
}

func (h *MessageHandler) handlePort(session *domain.Session, msg *domain.MessageEvent) error {
	port := strings.TrimSpace(msg.Message)
	if len(port) == 0 {
		return h.sendMessage(msg.ChatID, "❌ Porta inválida. Por favor, digite uma porta válida:")
	}

	session.Port = port
	session.State = domain.StateConfirmData
	h.sessionService.UpdateSession(session)

	// Send confirmation
	keyboard := &domain.Keyboard{
		Inline: true,
		Buttons: [][]domain.Button{
			{
				{Text: "✅ Sim", Data: "confirm:yes"},
				{Text: "❌ Não", Data: "confirm:no"},
			},
		},
	}

	message := fmt.Sprintf(
		"📋 Confirme os dados da mudança:\n\n"+
			"📟 Serial Atual: %s\n"+
			"🌐 Nova OLT: %s\n"+
			"🔌 Slot: %s\n"+
			"🔌 Porta: %s\n\n"+
			"Os dados estão corretos?",
		session.OldSerialNumber,
		session.OLT,
		session.Slot,
		session.Port,
	)

	return h.sendMessageWithKeyboard(msg.ChatID, message, keyboard)
}

func (h *MessageHandler) handleConfirmation(session *domain.Session, confirm string) error {
	if confirm != "yes" {
		session.State = domain.StateServiceSelection
		h.sessionService.UpdateSession(session)
		return h.sendServiceMenu(session)
	}

	// Send typing action
	h.eventManager.MustFire("telegram.send.typing", event.M{"chatID": session.ChatID})

	// Send provisioning message
	h.sendMessage(session.ChatID, "⏳ Aguarde enquanto estamos provisionando o equipamento...")

	// Simulate provisioning
	time.Sleep(3 * time.Second)

	// Process based on service type
	var result string
	switch session.ServiceType {
	case domain.ServiceActivation:
		result = h.provisioningService.ActivateEquipment(session.Contract, session.SerialNumber)
	case domain.ServiceMaintenance:
		result = h.provisioningService.ReplaceEquipment(session.OldSerialNumber, session.SerialNumber, session.Contract)
	case domain.ServiceAddressChange:
		result = h.provisioningService.ChangeAddress(session.OldSerialNumber, session.OLT, session.Slot, session.Port)
	}

	// Reset session
	session.State = domain.StateIdle
	h.sessionService.UpdateSession(session)

	// Send result
	return h.sendMessage(session.ChatID, result)
}

func (h *MessageHandler) sendMessage(chatID int64, text string) error {
	response := &domain.MessageResponse{
		ChatID: chatID,
		Text:   text,
	}

	h.eventManager.MustFire("telegram.send.message", event.M{
		"response": response,
	})

	return nil
}

func (h *MessageHandler) sendMessageWithKeyboard(chatID int64, text string, keyboard *domain.Keyboard) error {
	response := &domain.MessageResponse{
		ChatID:   chatID,
		Text:     text,
		Keyboard: keyboard,
	}

	h.eventManager.MustFire("telegram.send.message", event.M{
		"response": response,
	})

	return nil
}
