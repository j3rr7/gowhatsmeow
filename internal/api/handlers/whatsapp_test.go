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
)

type mockWhatsAppService struct {
	state     whatsapp.ConnectionState
	jid       string
	qrCode    string
	sendID    string
	sendErr   error
	logoutErr error
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
