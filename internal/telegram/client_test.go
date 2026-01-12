package telegram

import (
	"context"
	"telegraph_uploader_v2/internal/config"
	"testing"
	"time"

	"github.com/gotd/td/tg"
	"github.com/gotd/td/telegram"
)

func TestFilterAdminChannels(t *testing.T) {
	// Helper to create channel with admin rights
	createAdminChannel := func(id int64) *tg.Channel {
		c := &tg.Channel{ID: id, Left: false}
		c.SetAdminRights(tg.ChatAdminRights{PostMessages: true})
		return c
	}

	createCreatorChannel := func(id int64) *tg.Channel {
		c := &tg.Channel{ID: id, Creator: true, Left: false}
		return c
	}

	tests := []struct {
		name     string
		chats    []tg.ChatClass
		expected int
	}{
		{
			name: "Creator",
			chats: []tg.ChatClass{
				createCreatorChannel(1),
			},
			expected: 1,
		},
		{
			name: "Admin",
			chats: []tg.ChatClass{
				createAdminChannel(2),
			},
			expected: 1,
		},
		{
			name: "Member (Not Admin)",
			chats: []tg.ChatClass{
				&tg.Channel{ID: 3, Creator: false, AdminRights: tg.ChatAdminRights{}, Left: false},
			},
			expected: 0,
		},
		{
			name: "Left Channel",
			chats: []tg.ChatClass{
				&tg.Channel{ID: 4, Creator: true, Left: true},
			},
			expected: 0,
		},
		{
			name: "Mixed",
			chats: []tg.ChatClass{
				createCreatorChannel(1),
				&tg.Channel{ID: 3, Creator: false},
				createAdminChannel(2),
			},
			expected: 2,
		},
		{
			name:     "Non-Channel Chat",
			chats:    []tg.ChatClass{&tg.Chat{ID: 5}},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterAdminChannels(tt.chats)
			if len(result) != tt.expected {
				t.Errorf("expected %d channels, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		TelegramAppId:   12345,
		TelegramApiHash: "hashhash",
	}

	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if client.appID != 12345 {
		t.Errorf("expected AppID 12345, got %d", client.appID)
	}
	if client.apiHash != "hashhash" {
		t.Errorf("expected ApiHash hashhash, got %s", client.apiHash)
	}
}

func TestNewWithClient(t *testing.T) {
	// Just verify it wraps correctly
	rawClient := telegram.NewClient(1, "h", telegram.Options{})
	cfg := &config.Config{TelegramAppId: 1, TelegramApiHash: "h"}
	
	c := NewWithClient(rawClient, cfg)
	if c.client != rawClient {
		t.Error("client not stored")
	}
	if c.ready == nil {
		t.Error("ready chan not init")
	}
}

func TestClient_StartStop(t *testing.T) {
	// We can't easily test connection logic without a mock server,
	// but we can test that Start launches the goroutine and Stop cancels context.
	
	// Create a dummy client
	cfg := &config.Config{TelegramAppId: 1, TelegramApiHash: "h"}
	c, _ := New(cfg)

	ctx := context.Background()
	c.Start(ctx)
	
	if c.ctx == nil {
		t.Error("context not initialized")
	}
	
	// Stop immediately
	c.Stop()
	
	// Check cancellation
	select {
	case <-c.ctx.Done():
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Error("context not cancelled after Stop")
	}
	
	// Verify calling Start again does nothing
	oldCtx := c.ctx
	c.Start(ctx)
	if c.ctx != oldCtx {
		t.Error("Start should be idempotent/protected by lock for same instance lifecycle")
	}
}
