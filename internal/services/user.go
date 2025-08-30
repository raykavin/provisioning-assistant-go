package services

import (
	"provisioning-assistant/internal/domain"
	"strings"
	"time"
)

type UserService struct {
	// In production, this would connect to a database
	users map[string]*domain.User
}

func NewUserService() *UserService {
	// Mock data for testing
	users := map[string]*domain.User{
		"12345678901": {
			ID:        1,
			CPF:       "12345678901",
			Name:      "Raykavin Meireles",
			IsValid:   true,
			CreatedAt: time.Now(),
		},
		"98765432109": {
			ID:        2,
			CPF:       "98765432109",
			Name:      "Maria Santos",
			IsValid:   true,
			CreatedAt: time.Now(),
		},
		"11111111111": {
			ID:        3,
			CPF:       "11111111111",
			Name:      "Pedro Oliveira",
			IsValid:   true,
			CreatedAt: time.Now(),
		},
	}

	return &UserService{
		users: users,
	}
}

func (s *UserService) ValidateCPF(cpf string) *domain.User {
	cpf = strings.TrimSpace(cpf)

	// Simulate CPF validation
	if user, exists := s.users[cpf]; exists && user.IsValid {
		return user
	}

	return nil
}

func (s *UserService) GetUserByCPF(cpf string) *domain.User {
	if user, exists := s.users[cpf]; exists {
		return user
	}
	return nil
}
