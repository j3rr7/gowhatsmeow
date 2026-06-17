package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

type WhatsAppService interface {
	GetStatus() (ConnectionState, string, string) // state, jid, qrCode
	SendMessage(ctx context.Context, to string, text string) (string, error)
	SendFile(ctx context.Context, to string, fileData []byte, fileName string, mimeType string, caption string) (string, error)
	Logout(ctx context.Context) error
}

type whatsappService struct {
	manager *WhatsAppManager
}

func NewWhatsAppService(manager *WhatsAppManager) WhatsAppService {
	return &whatsappService{
		manager: manager,
	}
}

func (s *whatsappService) GetStatus() (ConnectionState, string, string) {
	s.manager.stateMu.RLock()
	defer s.manager.stateMu.RUnlock()

	var jid string
	if s.manager.client != nil && s.manager.client.Store.ID != nil {
		jid = s.manager.client.Store.ID.String()
	}

	return s.manager.state, jid, s.manager.qrCode
}

func (s *whatsappService) SendMessage(ctx context.Context, to string, text string) (string, error) {
	if s.manager.client == nil {
		return "", errors.New("whatsapp client is not initialized")
	}

	s.manager.stateMu.RLock()
	connected := s.manager.state == StateConnected
	s.manager.stateMu.RUnlock()

	if !connected {
		return "", errors.New("whatsapp client is not connected")
	}

	jid, err := types.ParseJID(to)
	if err != nil {
		jid, err = types.ParseJID(to + "@s.whatsapp.net")
		if err != nil {
			return "", fmt.Errorf("invalid recipient address: %w", err)
		}
	}

	msg := &waE2E.Message{
		Conversation: proto.String(text),
	}

	resp, err := s.manager.client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to dispatch message: %w", err)
	}

	return resp.ID, nil
}

func (s *whatsappService) SendFile(ctx context.Context, to string, fileData []byte, fileName string, mimeType string, caption string) (string, error) {
	if s.manager.client == nil {
		return "", errors.New("whatsapp client is not initialized")
	}

	s.manager.stateMu.RLock()
	connected := s.manager.state == StateConnected
	s.manager.stateMu.RUnlock()

	if !connected {
		return "", errors.New("whatsapp client is not connected")
	}

	jid, err := types.ParseJID(to)
	if err != nil {
		jid, err = types.ParseJID(to + "@s.whatsapp.net")
		if err != nil {
			return "", fmt.Errorf("invalid recipient address: %w", err)
		}
	}

	// Auto-detect media type
	var mediaType whatsmeow.MediaType
	if strings.HasPrefix(mimeType, "image/") {
		mediaType = whatsmeow.MediaImage
	} else if strings.HasPrefix(mimeType, "audio/") {
		mediaType = whatsmeow.MediaAudio
	} else if strings.HasPrefix(mimeType, "video/") {
		mediaType = whatsmeow.MediaVideo
	} else {
		mediaType = whatsmeow.MediaDocument
	}

	resp, err := s.manager.client.Upload(ctx, fileData, mediaType)
	if err != nil {
		return "", fmt.Errorf("failed to upload media: %w", err)
	}

	msg := &waE2E.Message{}
	switch mediaType {
	case whatsmeow.MediaImage:
		msg.ImageMessage = &waE2E.ImageMessage{
			URL:           proto.String(resp.URL),
			DirectPath:    proto.String(resp.DirectPath),
			MediaKey:      resp.MediaKey,
			Mimetype:      proto.String(mimeType),
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(fileData))),
			Caption:       proto.String(caption),
		}
	case whatsmeow.MediaAudio:
		msg.AudioMessage = &waE2E.AudioMessage{
			URL:           proto.String(resp.URL),
			DirectPath:    proto.String(resp.DirectPath),
			MediaKey:      resp.MediaKey,
			Mimetype:      proto.String(mimeType),
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(fileData))),
		}
	case whatsmeow.MediaVideo:
		msg.VideoMessage = &waE2E.VideoMessage{
			URL:           proto.String(resp.URL),
			DirectPath:    proto.String(resp.DirectPath),
			MediaKey:      resp.MediaKey,
			Mimetype:      proto.String(mimeType),
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(fileData))),
			Caption:       proto.String(caption),
		}
	default:
		msg.DocumentMessage = &waE2E.DocumentMessage{
			URL:           proto.String(resp.URL),
			DirectPath:    proto.String(resp.DirectPath),
			MediaKey:      resp.MediaKey,
			Mimetype:      proto.String(mimeType),
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(fileData))),
			Title:         proto.String(fileName),
			FileName:      proto.String(fileName),
		}
	}

	sendResp, err := s.manager.client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to dispatch media message: %w", err)
	}

	return sendResp.ID, nil
}

func (s *whatsappService) Logout(ctx context.Context) error {
	if s.manager.client == nil {
		return errors.New("whatsapp client is not initialized")
	}

	err := s.manager.client.Logout(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute logout: %w", err)
	}

	s.manager.stateMu.Lock()
	s.manager.state = StateDisconnected
	s.manager.qrCode = ""
	s.manager.stateMu.Unlock()

	return nil
}
