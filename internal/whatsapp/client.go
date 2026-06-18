package whatsapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"gowhatsmeow/internal/config"
	"gowhatsmeow/internal/logger"
	"net/http"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ConnectionState string

const (
	StateDisconnected ConnectionState = "disconnected"
	StateConnecting   ConnectionState = "connecting"
	StateQRReady      ConnectionState = "qr_ready"
	StateConnected    ConnectionState = "connected"
)

type WhatsAppManager struct {
	client    *whatsmeow.Client
	container *sqlstore.Container
	logger    waLog.Logger
	sugar     *zap.SugaredLogger

	stateMu sync.RWMutex
	state   ConnectionState
	qrCode  string

	qrDoneChan chan struct{}

	// Webhook Configuration
	webhookURLMu sync.RWMutex
	webhookURL   string
	httpClient   *http.Client

	// Safety Settings
	safetyCfg config.SafetyConfig

	// Duplicate cache
	recentSentMu sync.Mutex
	recentSent   map[string][]time.Time
}

func NewWhatsAppManager(lc fx.Lifecycle, container *sqlstore.Container, cfg *config.Config, zapLog *zap.Logger) *WhatsAppManager {
	waLogAdapter := logger.NewZapWAAdapter(zapLog)
	mgr := &WhatsAppManager{
		container:  container,
		logger:     waLogAdapter,
		sugar:      zapLog.Sugar().With("module", "whatsapp-manager"),
		state:      StateDisconnected,
		qrDoneChan: make(chan struct{}),
		webhookURL: cfg.Webhook.URL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		safetyCfg:  cfg.Safety,
		recentSent: make(map[string][]time.Time),
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return mgr.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return mgr.Stop(ctx)
		},
	})

	return mgr
}

func (m *WhatsAppManager) Start(ctx context.Context) error {
	m.sugar.Info("Starting WhatsApp manager...")

	deviceStore, err := m.container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get device store: %w", err)
	}

	clientLog := m.logger.Sub("Client")
	m.client = whatsmeow.NewClient(deviceStore, clientLog)
	m.client.AddEventHandler(m.handleEvent)

	if m.client.Store.ID == nil {
		m.sugar.Info("No WhatsApp session found. Listening to QR channel for authentication...")
		qrChan, err := m.client.GetQRChannel(context.Background())
		if err != nil {
			return fmt.Errorf("failed to get QR channel: %w", err)
		}

		err = m.client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}

		go m.listenToQR(qrChan)
	} else {
		m.sugar.Info("Existing session found. Reconnecting to WhatsApp...")
		m.stateMu.Lock()
		m.state = StateConnecting
		m.stateMu.Unlock()

		err = m.client.Connect()
		if err != nil {
			return fmt.Errorf("failed to reconnect: %w", err)
		}
	}

	return nil
}

func (m *WhatsAppManager) Stop(ctx context.Context) error {
	m.sugar.Info("Stopping WhatsApp manager...")
	close(m.qrDoneChan)
	if m.client != nil {
		m.client.Disconnect()
	}
	return nil
}

func (m *WhatsAppManager) listenToQR(qrChan <-chan whatsmeow.QRChannelItem) {
	m.stateMu.Lock()
	m.state = StateQRReady
	m.stateMu.Unlock()

	defer func() {
		m.stateMu.Lock()
		m.qrCode = ""
		if m.state != StateConnected {
			m.state = StateDisconnected
		}
		m.stateMu.Unlock()
	}()

	for {
		select {
		case evt, ok := <-qrChan:
			if !ok {
				m.sugar.Info("QR channel closed")
				return
			}
			switch evt.Event {
			case "code":
				m.stateMu.Lock()
				m.qrCode = evt.Code
				m.state = StateQRReady
				m.stateMu.Unlock()
				m.sugar.Info("New WhatsApp login QR code generated")
			case "success":
				m.stateMu.Lock()
				m.state = StateConnected
				m.qrCode = ""
				m.stateMu.Unlock()
				m.sugar.Info("Successfully logged in to WhatsApp")
				return
			case "timeout":
				m.stateMu.Lock()
				m.state = StateDisconnected
				m.qrCode = ""
				m.stateMu.Unlock()
				m.sugar.Warn("WhatsApp QR generation timed out")
				return
			}
		case <-m.qrDoneChan:
			m.sugar.Debug("QR listener stopped by lifecycle")
			return
		}
	}
}

func (m *WhatsAppManager) handleEvent(evt any) {
	switch v := evt.(type) {
	case *events.Connected:
		m.stateMu.Lock()
		m.state = StateConnected
		m.stateMu.Unlock()
		m.sugar.Info("Connected to WhatsApp socket")
	case *events.Disconnected:
		m.stateMu.Lock()
		if m.state == StateConnected {
			m.state = StateConnecting
		}
		m.stateMu.Unlock()
		m.sugar.Warn("Disconnected from WhatsApp socket")
	case *events.LoggedOut:
		m.stateMu.Lock()
		m.state = StateDisconnected
		m.qrCode = ""
		m.stateMu.Unlock()
		m.sugar.Info("Logged out of WhatsApp session")
	case *events.Message:
		m.sugar.Infof("Message from %s: %s", v.Info.Sender, v.Message.GetConversation())
		msgType, msgText, mediaData := parseMessageContent(v.Message)
		payload := map[string]interface{}{
			"event":     "message",
			"timestamp": v.Info.Timestamp.Unix(),
			"data": map[string]interface{}{
				"message_id": v.Info.ID,
				"sender":     v.Info.Sender.String(),
				"chat":       v.Info.Chat.String(),
				"is_from_me": v.Info.IsFromMe,
				"is_group":   v.Info.IsGroup,
				"type":       msgType,
				"text":       msgText,
				"media":      mediaData,
			},
		}
		m.dispatchWebhook(payload)
	case *events.Receipt:
		statusStr := parseReceiptType(v.Type)
		var msgIDs []string
		for _, mid := range v.MessageIDs {
			msgIDs = append(msgIDs, string(mid))
		}
		payload := map[string]interface{}{
			"event":     "message_status",
			"timestamp": v.Timestamp.Unix(),
			"data": map[string]interface{}{
				"message_ids": msgIDs,
				"chat":        v.Chat.String(),
				"sender":      v.Sender.String(),
				"status":      statusStr,
			},
		}
		m.dispatchWebhook(payload)
	}
}

func (m *WhatsAppManager) GetWebhookURL() string {
	m.webhookURLMu.RLock()
	defer m.webhookURLMu.RUnlock()
	return m.webhookURL
}

func (m *WhatsAppManager) SetWebhookURL(url string) {
	m.webhookURLMu.Lock()
	m.webhookURL = url
	m.webhookURLMu.Unlock()
	m.sugar.Infof("Webhook URL updated dynamically to: %s", url)
}

func (m *WhatsAppManager) CheckDuplicate(text string) error {
	if m.safetyCfg.DuplicateWindowSec <= 0 || m.safetyCfg.MaxDuplicateSends <= 0 {
		return nil
	}

	hash := getMessageHash(text)
	now := time.Now()
	window := time.Duration(m.safetyCfg.DuplicateWindowSec) * time.Second

	m.recentSentMu.Lock()
	defer m.recentSentMu.Unlock()

	timestamps := m.recentSent[hash]
	var activeTimestamps []time.Time
	for _, t := range timestamps {
		if now.Sub(t) < window {
			activeTimestamps = append(activeTimestamps, t)
		}
	}
	m.recentSent[hash] = activeTimestamps

	if len(activeTimestamps) >= m.safetyCfg.MaxDuplicateSends {
		return fmt.Errorf("duplicate message blocked: identical message sent %d times within the last %d seconds", len(activeTimestamps), m.safetyCfg.DuplicateWindowSec)
	}

	return nil
}

func (m *WhatsAppManager) RecordSentMessage(text string) {
	if m.safetyCfg.DuplicateWindowSec <= 0 || m.safetyCfg.MaxDuplicateSends <= 0 {
		return
	}

	hash := getMessageHash(text)
	now := time.Now()

	m.recentSentMu.Lock()
	defer m.recentSentMu.Unlock()

	m.recentSent[hash] = append(m.recentSent[hash], now)
}

func (m *WhatsAppManager) dispatchWebhook(payload interface{}) {
	m.webhookURLMu.RLock()
	url := m.webhookURL
	m.webhookURLMu.RUnlock()

	if url == "" {
		return
	}

	go func() {
		data, err := json.Marshal(payload)
		if err != nil {
			m.sugar.Errorf("Failed to marshal webhook payload: %v", err)
			return
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
		if err != nil {
			m.sugar.Errorf("Failed to create webhook request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "gowhatsmeow-webhook/1.0")

		resp, err := m.httpClient.Do(req)
		if err != nil {
			m.sugar.Errorf("Webhook delivery to %s failed: %v", url, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			m.sugar.Warnf("Webhook delivery to %s returned status %s", url, resp.Status)
		}
	}()
}

func getMessageHash(text string) string {
	h := sha256.New()
	h.Write([]byte(text))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func parseReceiptType(t types.ReceiptType) string {
	switch string(t) {
	case "read":
		return "read"
	case "delivery", "delivered":
		return "delivered"
	case "played":
		return "played"
	default:
		return "sent"
	}
}

func parseMessageContent(msg *waE2E.Message) (string, string, map[string]interface{}) {
	if msg == nil {
		return "other", "", nil
	}

	if msg.Conversation != nil {
		return "text", msg.GetConversation(), nil
	}
	if msg.ExtendedTextMessage != nil {
		return "text", msg.ExtendedTextMessage.GetText(), nil
	}

	if msg.ImageMessage != nil {
		img := msg.ImageMessage
		media := map[string]interface{}{
			"mime_type":   img.GetMimetype(),
			"file_length": img.GetFileLength(),
			"caption":     img.GetCaption(),
		}
		return "image", img.GetCaption(), media
	}

	if msg.AudioMessage != nil {
		aud := msg.AudioMessage
		media := map[string]interface{}{
			"mime_type":   aud.GetMimetype(),
			"file_length": aud.GetFileLength(),
		}
		return "audio", "", media
	}

	if msg.VideoMessage != nil {
		vid := msg.VideoMessage
		media := map[string]interface{}{
			"mime_type":   vid.GetMimetype(),
			"file_length": vid.GetFileLength(),
			"caption":     vid.GetCaption(),
		}
		return "video", vid.GetCaption(), media
	}

	if msg.DocumentMessage != nil {
		doc := msg.DocumentMessage
		media := map[string]interface{}{
			"mime_type":   doc.GetMimetype(),
			"file_length": doc.GetFileLength(),
			"filename":    doc.GetFileName(),
			"title":       doc.GetTitle(),
		}
		return "document", doc.GetTitle(), media
	}

	return "other", "", nil
}

