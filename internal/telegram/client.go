package telegram

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"telegraph_uploader_v2/internal/config"
	"time"

	"github.com/gotd/td/clock"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/html"

	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
	"go.uber.org/zap"
	"rsc.io/qr"
)

type Client struct {
	client      *telegram.Client
	sender      *message.Sender
	apiHash     string
	appID       int
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.Mutex
	dispatcher  tg.UpdateDispatcher
	ready       chan struct{}
	isConnected atomic.Bool
}

func New(cfg *config.Config) (*Client, error) {
	log.Printf("[Telegram] App ID: %d, API Hash: %s", cfg.TelegramAppId, cfg.TelegramApiHash)
	sessionDir, _ := os.Getwd()
	sessionPath := filepath.Join(sessionDir, "telegram-session.json")
	log.Printf("[Telegram] Session path: %s", sessionPath)

	// Set logging to WarnLevel to see important issues but avoid spam
	zCfg := zap.NewProductionConfig()
	zCfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	logger, _ := zCfg.Build()

	// Sync time with Google to handle system clock skew
	offset := syncTime()

	dispatcher := tg.NewUpdateDispatcher()

	opts := telegram.Options{
		Logger: logger,
		SessionStorage: &telegram.FileSessionStorage{
			Path: sessionPath,
		},
		Clock:         &OffsetClock{offset: offset},
		UpdateHandler: dispatcher,
	}

	client := telegram.NewClient(cfg.TelegramAppId, cfg.TelegramApiHash, opts)

	return &Client{
		client:     client,
		apiHash:    cfg.TelegramApiHash,
		appID:      cfg.TelegramAppId,
		ready:      make(chan struct{}),
		dispatcher: dispatcher,
	}, nil
}

func (c *Client) Start(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If already started, do nothing (optional safety)
	if c.ctx != nil {
		return
	}

	c.ctx, c.cancel = context.WithCancel(ctx)

	go func() {
		log.Println("[Telegram] Client background loop started")
		defer log.Println("[Telegram] Client background loop stopped")

		// Retry loop to ensure client stays alive
		for {
			select {
			case <-c.ctx.Done():
				return
			default:
				// Attempt to run the client
				log.Println("[Telegram] Connecting...")
				err := c.client.Run(c.ctx, func(ctx context.Context) error {
					// This callback is called ONCE when connected
					log.Println("[Telegram] Connected!")
					c.isConnected.Store(true)
					defer c.isConnected.Store(false)

					// Initialize sender
					c.sender = message.NewSender(tg.NewClient(c.client))

					// Check initial auth status
					status, err := c.client.Auth().Status(ctx)
					if err != nil {
						log.Println("[Telegram] Auth status check error:", err)
					} else if status.Authorized {
						log.Println("[Telegram] Authorized as user")
					} else {
						log.Println("[Telegram] Connected but NOT authorized. Login needed.")
					}

					// Signal ready (non-blocking)
					select {
					case c.ready <- struct{}{}:
					default:
					}

					// Block until context dies
					<-ctx.Done()
					return nil
				})

				// If Run returns, it means the connection was lost or closed
				if err != nil {
					// Check if it's just a context cancel
					if c.ctx.Err() != nil {
						return
					}
					log.Printf("[Telegram] Connection lost: %v. Reconnecting in 2s...", err)
					time.Sleep(2 * time.Second)
				} else {
					// Run returned nil, usually means clean shutdown requested
					return
				}
			}
		}
	}()
}

func (c *Client) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
}

// AuthFlowHandler defines the interface for UI interaction
type AuthFlowHandler interface {
	GetCode(ctx context.Context, sentCode *tg.AuthSentCode) (string, error)
	GetPassword(ctx context.Context) (string, error)
}

// terminalAuthenticator manually implements auth.UserAuthenticator
type terminalAuthenticator struct {
	phone       string
	flowHandler AuthFlowHandler
}

func (a terminalAuthenticator) Phone(_ context.Context) (string, error) {
	return a.phone, nil
}

func (a terminalAuthenticator) Password(ctx context.Context) (string, error) {
	return a.flowHandler.GetPassword(ctx)
}

func (a terminalAuthenticator) Code(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
	return a.flowHandler.GetCode(ctx, sentCode)
}

func (a terminalAuthenticator) AcceptTermsOfService(_ context.Context, tos tg.HelpTermsOfService) error {
	return nil
}

func (a terminalAuthenticator) SignUp(_ context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, fmt.Errorf("signing up is not supported")
}

func (c *Client) Login(ctx context.Context, phone string, flowHandler AuthFlowHandler) error {
	// Wait for connection before attempting login
	if err := c.WaitForConnection(ctx); err != nil {
		return fmt.Errorf("failed to wait for connection: %w", err)
	}

	// Create our manual authenticator
	authenticator := terminalAuthenticator{
		phone:       phone,
		flowHandler: flowHandler,
	}

	// Create the Auth Flow using the struct
	flow := auth.NewFlow(
		authenticator,
		auth.SendCodeOptions{},
	)

	// We simply use the context passed. If c.client isn't running, this will fail.
	// The retry loop in Start should ensure it's running.
	return c.client.Auth().IfNecessary(ctx, flow)
}

// LoginQR starts the QR code login flow
// displayQR is a callback that receives the PNG bytes of the QR code
func (c *Client) LoginQR(ctx context.Context, displayQR func(qrImage []byte), flowHandler AuthFlowHandler) error {
	// Wait for connection
	if err := c.WaitForConnection(ctx); err != nil {
		return fmt.Errorf("failed to wait for connection: %w", err)
	}

	loggedIn := qrlogin.OnLoginToken(c.dispatcher)

	_, err := c.client.QR().Auth(ctx, loggedIn, func(ctx context.Context, token qrlogin.Token) error {
		log.Printf("[Telegram] QR Code token received: %s...", token.URL())

		code, err := qr.Encode(token.URL(), qr.L)
		if err != nil {
			return fmt.Errorf("failed to encode QR: %w", err)
		}

		displayQR(code.PNG())
		return nil
	})

	if err != nil {
		if tgerr.Is(err, "SESSION_PASSWORD_NEEDED") {
			log.Println("[Telegram] 2FA Password needed for QR login")

			// Request password from UI
			pwd, err := flowHandler.GetPassword(ctx)
			if err != nil {
				return fmt.Errorf("failed to get password: %w", err)
			}

			// Submit password
			_, err = c.client.Auth().Password(ctx, pwd)
			if err != nil {
				return fmt.Errorf("failed to submit 2FA password: %w", err)
			}
		} else {
			return fmt.Errorf("QR login failed: %w", err)
		}
	}

	status, err := c.client.Auth().Status(ctx)
	if err != nil {
		return fmt.Errorf("failed to check status after QR login: %w", err)
	}
	if !status.Authorized {
		return fmt.Errorf("QR login claimed success but user is not authorized")
	}

	// User is *tg.User because we are Authorized
	log.Printf("[Telegram] QR Login successful as %s", status.User.Username)
	return nil
}

// CheckAuth checks if the user is currently authorized
func (c *Client) CheckAuth(ctx context.Context) (bool, error) {
	status, err := c.client.Auth().Status(ctx)
	if err != nil {
		return false, err
	}
	return status.Authorized, nil
}

// WaitForConnection blocks until the client is connected or the context is cancelled.
// It returns nil if connected, or an error if the context is cancelled or times out.
func (c *Client) WaitForConnection(ctx context.Context) error {
	if c.isConnected.Load() {
		return nil
	}

	log.Println("[Telegram] Waiting for connection...")

	// Create a timeout context to prevent indefinite hanging
	waitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			if waitCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("connection timed out after 30s")
			}
			return waitCtx.Err()
		case <-ticker.C:
			if c.isConnected.Load() {
				log.Println("[Telegram] Connection established, proceeding...")
				return nil
			}
		}
	}
}

// OffsetClock implements telegram.Clock but adds an offset to the system time
type OffsetClock struct {
	offset time.Duration
}

func (c *OffsetClock) Now() time.Time {
	return time.Now().Add(c.offset)
}

func (c *OffsetClock) Timer(d time.Duration) clock.Timer {
	return &offsetTimer{t: time.NewTimer(d)}
}

func (c *OffsetClock) Ticker(d time.Duration) clock.Ticker {
	return &offsetTicker{t: time.NewTicker(d)}
}

type offsetTimer struct {
	t *time.Timer
}

func (t *offsetTimer) C() <-chan time.Time   { return t.t.C }
func (t *offsetTimer) Reset(d time.Duration) { t.t.Reset(d) }
func (t *offsetTimer) Stop() bool            { return t.t.Stop() }

type offsetTicker struct {
	t *time.Ticker
}

func (t *offsetTicker) C() <-chan time.Time   { return t.t.C }
func (t *offsetTicker) Stop()                 { t.t.Stop() }
func (t *offsetTicker) Reset(d time.Duration) { t.t.Reset(d) }

func syncTime() time.Duration {
	// Try to get time from google.com
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Head("https://www.google.com")
	if err != nil {
		log.Printf("[Telegram] Time sync failed: %v", err)
		return 0
	}
	defer resp.Body.Close()

	if dateStr := resp.Header.Get("Date"); dateStr != "" {
		if t, err := time.Parse(time.RFC1123, dateStr); err == nil {
			localTime := time.Now()
			offset := t.Sub(localTime)
			log.Printf("[Telegram] Time synced. System: %v, Network: %v, Offset: %v", localTime, t, offset)
			return offset
		}
	}
	return 0
}

// SearchAdminChannels searches for channels by query (min 3 chars) and returns those where the user is an admin.
// Limit is set to 10 as requested.
func (c *Client) SearchAdminChannels(ctx context.Context, query string) ([]*tg.Channel, error) {
	if len(query) < 3 {
		return nil, fmt.Errorf("query too short, must be at least 3 characters")
	}

	// Wait for connection
	if err := c.WaitForConnection(ctx); err != nil {
		return nil, fmt.Errorf("failed to wait for connection: %w", err)
	}

	// Search for contacts/chats with a limit of 10
	found, err := c.client.API().ContactsSearch(ctx, &tg.ContactsSearchRequest{
		Q:     query,
		Limit: 10,
	})
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var adminChannels []*tg.Channel

	// ContactsSearch returns *tg.ContactsFound which contains Chats []ChatClass
	// We need to iterate over Chats and filter for channels where we are admin.
	for _, chat := range found.Chats {
		channel, ok := chat.(*tg.Channel)
		if !ok {
			continue
		}

		// Filter out if we left the channel
		if channel.Left {
			continue
		}

		// Check if we are creator or have admin rights
		// AdminRights is an optional field.
		_, hasAdminRights := channel.GetAdminRights()

		if channel.Creator || hasAdminRights {
			adminChannels = append(adminChannels, channel)
		}
	}

	return adminChannels, nil
}

// ScheduleMessage schedules a message to be sent to a channel at a specific time.
func (c *Client) ScheduleMessage(ctx context.Context, channel *tg.Channel, text string, schedule time.Time) error {
	// Wait for connection
	if err := c.WaitForConnection(ctx); err != nil {
		return fmt.Errorf("failed to wait for connection: %w", err)
	}

	// Create input peer from channel
	inputPeer := &tg.InputPeerChannel{
		ChannelID:  channel.ID,
		AccessHash: channel.AccessHash,
	}

	// Send scheduled message
	_, err := c.sender.To(inputPeer).Schedule(schedule).StyledText(ctx, html.String(nil, text))
	if err != nil {
		return fmt.Errorf("failed to schedule message: %w", err)
	}

	return nil
}

// ScheduleMessageByID schedules a message using ID and AccessHash directly
func (c *Client) ScheduleMessageByID(ctx context.Context, channelID int64, accessHash int64, text string, schedule time.Time) error {
	// Wait for connection
	if err := c.WaitForConnection(ctx); err != nil {
		return fmt.Errorf("failed to wait for connection: %w", err)
	}

	// Create input peer
	inputPeer := &tg.InputPeerChannel{
		ChannelID:  channelID,
		AccessHash: accessHash,
	}

	// Send scheduled message
	_, err := c.sender.To(inputPeer).Schedule(schedule).StyledText(ctx, html.String(nil, text))
	if err != nil {
		return fmt.Errorf("failed to schedule message: %w", err)
	}

	return nil
}

// TelegramUser represents the logged-in user
type TelegramUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Photo     []byte `json:"photo"` // Base64 encoded in JSON
}

// GetMe returns the current authorized user
func (c *Client) GetMe(ctx context.Context) (*TelegramUser, error) {
	// Wait for connection
	if err := c.WaitForConnection(ctx); err != nil {
		return nil, fmt.Errorf("failed to wait for connection: %w", err)
	}

	// Get self
	user, err := c.client.Self(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get self: %w", err)
	}

	tgUser := &TelegramUser{
		ID:        user.ID,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}

	return tgUser, nil
}
