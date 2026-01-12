package telegram

import (
	"context"
)

// MockAuthFlowHandler
type MockAuthFlowHandler struct {
	Password string
	Err      error
}

func (m *MockAuthFlowHandler) GetPassword(ctx context.Context) (string, error) {
	return m.Password, m.Err
}

// Tests for terminalAuthenticator were removed as the struct was removed.