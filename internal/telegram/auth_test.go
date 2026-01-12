package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/gotd/td/tg"
)

// MockAuthFlowHandler
type MockAuthFlowHandler struct {
	Code     string
	Password string
	Err      error
}

func (m *MockAuthFlowHandler) GetCode(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
	return m.Code, m.Err
}

func (m *MockAuthFlowHandler) GetPassword(ctx context.Context) (string, error) {
	return m.Password, m.Err
}

func TestTerminalAuthenticator(t *testing.T) {
	mock := &MockAuthFlowHandler{
		Code:     "12345",
		Password: "pass",
	}
	
	auth := terminalAuthenticator{
		phone:       "555",
		flowHandler: mock,
	}

	ctx := context.Background()

	// Test Phone
	phone, err := auth.Phone(ctx)
	if err != nil {
		t.Errorf("Phone error: %v", err)
	}
	if phone != "555" {
		t.Errorf("expected phone 555, got %s", phone)
	}

	// Test Code
	code, err := auth.Code(ctx, &tg.AuthSentCode{})
	if err != nil {
		t.Errorf("Code error: %v", err)
	}
	if code != "12345" {
		t.Errorf("expected code 12345, got %s", code)
	}

	// Test Password
	pwd, err := auth.Password(ctx)
	if err != nil {
		t.Errorf("Password error: %v", err)
	}
	if pwd != "pass" {
		t.Errorf("expected password pass, got %s", pwd)
	}

	// Test AcceptTermsOfService
	err = auth.AcceptTermsOfService(ctx, tg.HelpTermsOfService{})
	if err != nil {
		t.Errorf("AcceptTermsOfService error: %v", err)
	}

	// Test SignUp
	_, err = auth.SignUp(ctx)
	if err == nil {
		t.Error("expected error for SignUp")
	}
}

func TestTerminalAuthenticator_Error(t *testing.T) {
	mock := &MockAuthFlowHandler{
		Err: errors.New("fail"),
	}
	auth := terminalAuthenticator{flowHandler: mock}

	_, err := auth.Code(context.Background(), nil)
	if err == nil {
		t.Error("expected error for Code")
	}

	_, err = auth.Password(context.Background())
	if err == nil {
		t.Error("expected error for Password")
	}
}
