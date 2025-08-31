package services

import (
	"context"
	"fmt"
	"provisioning-assistant/internal/domain"
	"provisioning-assistant/internal/domain/dto"
	"provisioning-assistant/internal/unm"
	"strconv"
	"strings"
)

type ProvisioningService struct {
	unmClient *unm.UNMClient
	logger    domain.Logger
}

// NewProvisioningService creates a new provisioning service instance
func NewProvisioningService(unmClient *unm.UNMClient, logger domain.Logger) *ProvisioningService {
	return &ProvisioningService{
		unmClient: unmClient,
		logger:    logger,
	}
}

// ProvisionEquipment provisions an ONU equipment and returns signal information
func (s *ProvisioningService) ProvisionEquipment(ctx context.Context, connInfo *dto.ConnectionInfo) (*domain.OnuSignalInfo, error) {
	if err := s.validateConnectionInfo(connInfo); err != nil {
		return nil, fmt.Errorf("informações de conexão inválidas: %w", err)
	}

	slot, port, err := s.parseOltSlotPort(connInfo.ConnectionOltSlot, connInfo.ConnectionOltPort)
	if err != nil {
		return nil, fmt.Errorf("falha ao analisar slot/porta da OLT: %w", err)
	}

	config := unm.OnuProvisioningConfig{
		PonSlot:      slot,
		PonPort:      port,
		ClientName:   connInfo.ClientName,
		OltIP:        connInfo.ConnectionOltIP,
		Vlan:         connInfo.ConnectionClientVlan,
		PPPoEUser:    connInfo.ConnectionClientPPPoEUsername,
		PPPoEPass:    connInfo.ConnectionClientPPPoEPassword,
		Serial:       connInfo.ConnectionEquipmentSerialNumber,
		SplitterName: connInfo.ConnectionClientSplitterName,
		SplitterPort: connInfo.ConnectionClientSplitterPort,
		Model:        "AN5506-01-A1",
	}

	s.logger.WithFields(map[string]any{
		"olt":       config.OltIP,
		"serial":    config.Serial,
		"cliente":   config.ClientName,
		"protocolo": connInfo.AssignmentErpID,
	}).Info("Iniciando provisionamento do equipamento")

	if err := s.unmClient.OnuProvisioning(ctx, config); err != nil {
		return nil, fmt.Errorf("falha no provisionamento: %w", err)
	}

	signalInfo, err := s.fetchOnuSignal(ctx, config)
	if err != nil {
		s.logger.WithError(err).Warn("Falha ao obter informações de sinal da ONU")
		return nil, nil
	}

	return signalInfo, nil
}

// fetchOnuSignal retrieves optical signal information from the ONU
func (s *ProvisioningService) fetchOnuSignal(ctx context.Context, config unm.OnuProvisioningConfig) (*domain.OnuSignalInfo, error) {
	opticalInfo, err := s.unmClient.OnuInfo(
		ctx,
		config.PonSlot,
		config.PonPort,
		config.OltIP,
		config.Serial,
	)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter informações ópticas: %w", err)
	}

	return &domain.OnuSignalInfo{
		TxPower: opticalInfo.TxPower,
		RxPower: opticalInfo.RxPower,
	}, nil
}

// validateConnectionInfo validates the connection information structure
func (s *ProvisioningService) validateConnectionInfo(connInfo *dto.ConnectionInfo) error {
	if connInfo == nil {
		return fmt.Errorf("informações de conexão são nulas")
	}
	if connInfo.ConnectionOltIP == "" {
		return fmt.Errorf("IP da OLT é obrigatório")
	}
	if connInfo.ConnectionEquipmentSerialNumber == "" {
		return fmt.Errorf("número de série do equipamento é obrigatório")
	}
	if connInfo.ConnectionClientPPPoEUsername == "" {
		return fmt.Errorf("nome de usuário PPPoE é obrigatório")
	}
	if connInfo.ConnectionClientPPPoEPassword == "" {
		return fmt.Errorf("senha PPPoE é obrigatória")
	}
	if connInfo.ConnectionClientVlan == "" {
		return fmt.Errorf("VLAN é obrigatória")
	}
	return nil
}

// parseOltSlotPort parses string slot and port values to unsigned integers
func (s *ProvisioningService) parseOltSlotPort(slotStr, portStr string) (uint, uint, error) {
	slot, err := strconv.ParseUint(strings.TrimSpace(slotStr), 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("slot inválido: %w", err)
	}

	port, err := strconv.ParseUint(strings.TrimSpace(portStr), 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("porta inválida: %w", err)
	}

	return uint(slot), uint(port), nil
}
