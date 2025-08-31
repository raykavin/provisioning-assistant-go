package handler

import "time"

// Message constants for the bot
const (
	// Welcome and authentication messages
	MSG_WELCOME = `Assistente de provisionamento - Fibralink
	Para continuar, preciso verificar sua identidade.
	Por favor, digite seu CPF (apenas números):`

	MSG_CPF_INVALID = "❌ CPF inválido. Digite apenas os 11 dígitos do CPF."

	MSG_CPF_UNAUTHORIZED = "❌ CPF não autorizado.\n" +
		"Por favor, verifique o número e tente novamente:"

	MSG_USER_GREETING = "✅ Olá, %s!\n\nO que você deseja fazer?"

	// Session messages
	MSG_SESSION_EXPIRED = "Sessão expirada. Por favor, digite /start para começar novamente."

	// Menu messages
	MSG_MENU_PROVISION = "🔧 Provisionar Equipamento"
	MSG_MENU_EXIT      = "❌ Sair"
	MSG_EXIT_MESSAGE   = "👋 Obrigado por usar nosso sistema. Até logo!"

	// Protocol messages
	MSG_REQUEST_PROTOCOL   = "📄 Por favor, informe o número do protocolo da solicitação:"
	MSG_PROTOCOL_INVALID   = "❌ Protocolo inválido. Por favor, digite apenas números:"
	MSG_SEARCHING_INFO     = "🔍 Buscando informações da solicitação..."
	MSG_PROTOCOL_NOT_FOUND = "❌ Não foi possível encontrar a solicitação.\n" +
		"Verifique o número do protocolo e tente novamente:"

	// Confirmation messages
	MSG_CONFIRM_DATA = "📋 Confirme os dados da solicitação:\n\n" +
		"📄 Contrato: %s\n" +
		"📝 Solicitação: %s\n" +
		"📟 Serial ONU: %s\n" +
		"🔲 CTO: %s\n" +
		"🔌 Porta CTO: %s\n\n" +
		"Você confirma os dados da solicitação?"

	MSG_CONFIRM_YES = "✅ Sim"
	MSG_CONFIRM_NO  = "❌ Não"

	MSG_CONFIRMATION_DENIED = "❌ Infelizmente não é possível continuar por aqui.\n\n" +
		"Por favor, entre em contato com o gerenciamento de campo para atualização das informações " +
		"ou provisionamento manual do equipamento."

	// Provisioning messages
	MSG_PROVISIONING_START = "⏳ Aguarde enquanto estamos provisionando o equipamento..."

	MSG_PROVISIONING_FAILED = "❌ Falha no provisionamento.\n\nErro: %v\n\n" +
		"Por favor, tente novamente ou entre em contato com o suporte."

	MSG_PROVISIONING_SUCCESS = "✅ Equipamento provisionado com sucesso!\n\n" +
		"📄 Contrato: %s\n" +
		"📟 Serial: %s\n" +
		"📶 Status: ONLINE\n"

	MSG_SIGNAL_INFO = "📡 Informações:\n" +
		"➡️ Pot. de recepção (dBm): %s dBm\n" +
		"⬅️ Pot. de transmissão (-dBm): %s dBm\n" +
		"🔋 Voltagem: %s V\n" +
		"🌡️ Temperatura: %s ºC\n"

	MSG_EQUIPMENT_READY = "\nO equipamento está pronto para uso!"
)

// Timeout constants
const (
	TIMEOUT_CPF_VALIDATION = 2 * time.Second
	TIMEOUT_ERP_FETCH      = 30 * time.Second
	TIMEOUT_PROVISIONING   = 60 * time.Second
)
