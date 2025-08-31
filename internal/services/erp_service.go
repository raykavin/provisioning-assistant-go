package services

import (
	"context"
	"fmt"
	"provisioning-assistant/internal/domain"
	"provisioning-assistant/internal/domain/dto"
)

type ErpService struct {
	repository domain.ErpRepository
	logger     domain.Logger
}

// NewErpService creates a new ERP service instance
func NewErpService(repository domain.ErpRepository, logger domain.Logger) *ErpService {
	return &ErpService{
		repository: repository,
		logger:     logger,
	}
}

// GetConnectionInfo retrieves connection information from ERP by protocol
func (s *ErpService) GetConnectionInfo(ctx context.Context, protocol string) (*dto.ConnectionInfo, error) {
	s.logger.WithField("protocol", protocol).Info("Buscando informações de conexão do ERP")

	connInfo, err := s.repository.GetConnInfoByProtocol(ctx, protocol)
	if err != nil {
		s.logger.WithError(err).WithField("protocol", protocol).Error("Falha ao buscar informações de conexão")
		return nil, fmt.Errorf("falha ao buscar informações de conexão: %w", err)
	}

	if connInfo.ConnectionOltIP == "" {
		return nil, fmt.Errorf("informações de conexão incompletas: IP da OLT ausente")
	}

	if connInfo.ConnectionEquipmentSerialNumber == "" {
		return nil, fmt.Errorf("informações de conexão incompletas: número de série do equipamento ausente")
	}

	s.logger.
		WithFields(map[string]any{
			"protocol": protocol,
			"contract": connInfo.ContractDescription,
			"olt_ip":   connInfo.ConnectionOltIP,
		}).Info("Informações de conexão obtidas com sucesso")

	return connInfo, nil
}
