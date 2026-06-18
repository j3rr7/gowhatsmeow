package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"gowhatsmeow/internal/whatsapp"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"go.mau.fi/whatsmeow/types"
)

type mockWhatsAppService struct {
	state      whatsapp.ConnectionState
	jid        string
	qrCode     string
	sendID     string
	sendErr    error
	logoutErr  error
	webhookURL string
	isOnWA     []types.IsOnWhatsAppResponse
	isOnWAErr  error
}

func (m *mockWhatsAppService) GetStatus() (whatsapp.ConnectionState, string, string) {
	return m.state, m.jid, m.qrCode
}

func (m *mockWhatsAppService) SendMessage(ctx context.Context, to string, text string) (string, error) {
	return m.sendID, m.sendErr
}

func (m *mockWhatsAppService) SendFile(ctx context.Context, to string, fileData []byte, fileName string, mimeType string, caption string) (string, error) {
	return m.sendID, m.sendErr
}

func (m *mockWhatsAppService) Logout(ctx context.Context) error {
	return m.logoutErr
}

func (m *mockWhatsAppService) IsOnWhatsApp(ctx context.Context, numbers []string) ([]types.IsOnWhatsAppResponse, error) {
	return m.isOnWA, m.isOnWAErr
}

func (m *mockWhatsAppService) GetWebhookURL() string {
	return m.webhookURL
}

func (m *mockWhatsAppService) SetWebhookURL(url string) {
	m.webhookURL = url
}

func TestGetStatus(t *testing.T) {
	e := echo.New()
	mockSvc := &mockWhatsAppService{
		state:  whatsapp.StateConnected,
		jid:    "123456789@s.whatsapp.net",
		qrCode: "",
	}
	h := NewWhatsAppHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/whatsapp/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.GetStatus(c); err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if resp["status"] != string(whatsapp.StateConnected) {
		t.Errorf("expected status connected, got %v", resp["status"])
	}
	if resp["jid"] != "123456789@s.whatsapp.net" {
		t.Errorf("expected jid, got %v", resp["jid"])
	}
}

func TestSendMessage(t *testing.T) {
	e := echo.New()
	mockSvc := &mockWhatsAppService{
		sendID: "ABC123XYZ",
	}
	h := NewWhatsAppHandler(mockSvc)

	reqBody := `{"to":"628123456789","message":"Hello!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/whatsapp/send", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.SendMessage(c); err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if resp["success"] != true {
		t.Errorf("expected success true, got %v", resp["success"])
	}
	if resp["message_id"] != "ABC123XYZ" {
		t.Errorf("expected message_id ABC123XYZ, got %v", resp["message_id"])
	}
}

func TestSendFile(t *testing.T) {
	e := echo.New()
	mockSvc := &mockWhatsAppService{
		sendID: "MEDIA123",
	}
	h := NewWhatsAppHandler(mockSvc)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	toField, _ := writer.CreateFormField("to")
	toField.Write([]byte("628123456789"))

	captionField, _ := writer.CreateFormField("caption")
	captionField.Write([]byte("My test file"))

	fileField, _ := writer.CreateFormFile("file", "test.txt")
	fileField.Write([]byte("file content goes here"))

	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/whatsapp/send-file", body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.SendFile(c); err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if resp["success"] != true {
		t.Errorf("expected success true, got %v", resp["success"])
	}
	if resp["message_id"] != "MEDIA123" {
		t.Errorf("expected message_id MEDIA123, got %v", resp["message_id"])
	}
}

func TestGetWebhook(t *testing.T) {
	e := echo.New()
	mockSvc := &mockWhatsAppService{
		webhookURL: "https://my-webhook.com",
	}
	h := NewWhatsAppHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/whatsapp/webhook", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.GetWebhook(c); err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if resp["url"] != "https://my-webhook.com" {
		t.Errorf("expected url https://my-webhook.com, got %v", resp["url"])
	}
}

func TestSetWebhook(t *testing.T) {
	e := echo.New()
	mockSvc := &mockWhatsAppService{}
	h := NewWhatsAppHandler(mockSvc)

	reqBody := `{"url":"https://new-webhook.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/whatsapp/webhook", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.SetWebhook(c); err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if resp["success"] != true {
		t.Errorf("expected success true, got %v", resp["success"])
	}
	if resp["url"] != "https://new-webhook.com" {
		t.Errorf("expected url https://new-webhook.com, got %v", resp["url"])
	}
	if mockSvc.webhookURL != "https://new-webhook.com" {
		t.Errorf("expected mockSvc.webhookURL to be updated, got %s", mockSvc.webhookURL)
	}
}

func TestCheckNumber(t *testing.T) {
	e := echo.New()
	mockSvc := &mockWhatsAppService{
		isOnWA: []types.IsOnWhatsAppResponse{
			{
				Query: "628123456789",
				IsIn:  true,
				JID:   types.NewJID("628123456789", types.DefaultUserServer),
			},
		},
	}
	h := NewWhatsAppHandler(mockSvc)

	reqBody := `{"number":"+62 812-3456-789"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/whatsapp/check-number", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.CheckNumber(c); err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if resp["registered"] != true {
		t.Errorf("expected registered true, got %v", resp["registered"])
	}
	if resp["number"] != "628123456789" {
		t.Errorf("expected number 628123456789, got %v", resp["number"])
	}
	if resp["jid"] != "628123456789@s.whatsapp.net" {
		t.Errorf("expected jid 628123456789@s.whatsapp.net, got %v", resp["jid"])
	}
}

