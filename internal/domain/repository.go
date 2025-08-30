package domain

import (
	"context"
	"provisioning-assistant/internal/domain/dto"
)

type ErpRepository interface {
	GetConnInfoByProtocol(ctx context.Context, protocol string) (*dto.ConnectionInfo, error)
}
