package services

import (
	"provisioning-assistant/internal/domain"
	"strings"
	"time"
)

type UserService struct {
	authorizedCPF string
}

// NewUserService creates a new user service instance with test authorization
func NewUserService() *UserService {
	return &UserService{
		authorizedCPF: "12345678901",
	}
}

// ValidateTaxID validates a CPF and returns user information if authorized
func (s *UserService) ValidateTaxID(taxID string) *domain.User {
	taxID = strings.TrimSpace(taxID)

	if taxID == s.authorizedCPF {
		return &domain.User{
			ID:        1,
			CPF:       taxID,
			Name:      "Raykavin Meireles",
			IsValid:   true,
			CreatedAt: time.Now(),
		}
	}

	return nil
}
