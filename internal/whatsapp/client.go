package whatsapp

import (
	"context"
	"fmt"
	"gowhatsmeow/internal/logger"
	"sync"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
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
}

func NewWhatsAppManager(lc fx.Lifecycle, container *sqlstore.Container, zapLog *zap.Logger) *WhatsAppManager {
	waLogAdapter := logger.NewZapWAAdapter(zapLog)
	mgr := &WhatsAppManager{
		container:  container,
		logger:     waLogAdapter,
		sugar:      zapLog.Sugar().With("module", "whatsapp-manager"),
		state:      StateDisconnected,
		qrDoneChan: make(chan struct{}),
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
	}
}
