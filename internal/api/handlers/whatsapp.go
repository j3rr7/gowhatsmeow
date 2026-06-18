package handlers

import (
	"fmt"
	"gowhatsmeow/internal/whatsapp"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
)

type WhatsAppHandler struct {
	service whatsapp.WhatsAppService
}

func NewWhatsAppHandler(service whatsapp.WhatsAppService) *WhatsAppHandler {
	return &WhatsAppHandler{
		service: service,
	}
}

type SendMessageRequest struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

type WebhookRequest struct {
	URL string `json:"url"`
}

type CheckNumberRequest struct {
	Number  string   `json:"number"`
	Numbers []string `json:"numbers"`
}

type CheckNumberResult struct {
	Number     string `json:"number"`
	Registered bool   `json:"registered"`
	JID        string `json:"jid"`
}

func (h *WhatsAppHandler) GetStatus(c echo.Context) error {
	state, jid, qrCode := h.service.GetStatus()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  state,
		"jid":     jid,
		"has_qr":  qrCode != "",
		"qr_code": qrCode,
	})
}

func (h *WhatsAppHandler) GetQR(c echo.Context) error {
	state, _, qrCode := h.service.GetStatus()

	acceptHeader := c.Request().Header.Get("Accept")
	wantsHTML := strings.Contains(acceptHeader, "text/html")

	if wantsHTML {
		switch state {
		case whatsapp.StateConnected:
			return c.HTML(http.StatusOK, htmlAlreadyConnected)
		case whatsapp.StateQRReady:
			if qrCode == "" {
				return c.HTML(http.StatusOK, fmt.Sprintf(htmlLoading, "Waiting for QR generation..."))
			}
			return c.HTML(http.StatusOK, fmt.Sprintf(htmlQRTemplate, url.QueryEscape(qrCode)))
		default:
			return c.HTML(http.StatusOK, fmt.Sprintf(htmlLoading, "Connecting to WhatsApp..."))
		}
	}

	if state == whatsapp.StateConnected {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":  state,
			"message": "Already connected",
		})
	}

	if qrCode == "" {
		return c.JSON(http.StatusAccepted, map[string]interface{}{
			"status":  state,
			"message": "QR code is not generated yet, please try again in a few seconds",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  state,
		"qr_code": qrCode,
	})
}

func (h *WhatsAppHandler) SendMessage(c echo.Context) error {
	var req SendMessageRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if strings.TrimSpace(req.To) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Recipient ('to') is required")
	}
	if strings.TrimSpace(req.Message) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Message body ('message') is required")
	}

	msgID, err := h.service.SendMessage(c.Request().Context(), req.To, req.Message)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":    true,
		"message_id": msgID,
	})
}

func (h *WhatsAppHandler) SendFile(c echo.Context) error {
	to := c.FormValue("to")
	caption := c.FormValue("caption")

	if strings.TrimSpace(to) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Recipient ('to') is required")
	}

	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "File ('file') is required")
	}

	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to open uploaded file")
	}
	defer src.Close()

	fileData, err := io.ReadAll(src)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to read uploaded file")
	}

	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	msgID, err := h.service.SendFile(c.Request().Context(), to, fileData, file.Filename, mimeType, caption)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":    true,
		"message_id": msgID,
	})
}

func (h *WhatsAppHandler) Logout(c echo.Context) error {
	err := h.service.Logout(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Logged out and session cleared successfully",
	})
}

func (h *WhatsAppHandler) GetWebhook(c echo.Context) error {
	url := h.service.GetWebhookURL()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"url": url,
	})
}

func (h *WhatsAppHandler) SetWebhook(c echo.Context) error {
	var req WebhookRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	h.service.SetWebhookURL(req.URL)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Webhook URL updated successfully",
		"url":     req.URL,
	})
}

func (h *WhatsAppHandler) CheckNumber(c echo.Context) error {
	var req CheckNumberRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	var numbersToCheck []string
	if req.Number != "" {
		numbersToCheck = append(numbersToCheck, req.Number)
	}
	for _, num := range req.Numbers {
		if num != "" {
			numbersToCheck = append(numbersToCheck, num)
		}
	}

	if len(numbersToCheck) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Either 'number' or 'numbers' array must be provided")
	}

	cleanedNumbers := make([]string, len(numbersToCheck))
	for i, num := range numbersToCheck {
		cleaned := strings.TrimSpace(num)
		cleaned = strings.ReplaceAll(cleaned, "+", "")
		cleaned = strings.ReplaceAll(cleaned, " ", "")
		cleaned = strings.ReplaceAll(cleaned, "-", "")
		cleanedNumbers[i] = cleaned
	}

	results, err := h.service.IsOnWhatsApp(c.Request().Context(), cleanedNumbers)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var responseResults []CheckNumberResult
	for _, res := range results {
		responseResults = append(responseResults, CheckNumberResult{
			Number:     res.Query,
			Registered: res.IsIn,
			JID:        res.JID.String(),
		})
	}

	var singleResult *CheckNumberResult
	if len(responseResults) == 1 {
		singleResult = &responseResults[0]
	}

	respPayload := map[string]interface{}{
		"results": responseResults,
	}
	if singleResult != nil {
		respPayload["registered"] = singleResult.Registered
		respPayload["jid"] = singleResult.JID
		respPayload["number"] = singleResult.Number
	}

	return c.JSON(http.StatusOK, respPayload)
}


const htmlQRTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>WhatsApp Login QR</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;600&display=swap" rel="stylesheet">
    <style>
        body {
            background: radial-gradient(circle at center, #1a1c24 0%%, #111218 100%%);
            color: #ffffff;
            font-family: 'Outfit', sans-serif;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            margin: 0;
            padding: 20px;
            box-sizing: border-box;
        }
        .container {
            background: rgba(255, 255, 255, 0.03);
            backdrop-filter: blur(20px);
            -webkit-backdrop-filter: blur(20px);
            border: 1px solid rgba(255, 255, 255, 0.08);
            padding: 40px;
            border-radius: 24px;
            box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5);
            text-align: center;
            max-width: 420px;
            width: 100%%;
            animation: fadeIn 0.8s ease-out;
        }
        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(20px); }
            to { opacity: 1; transform: translateY(0); }
        }
        h1 {
            font-size: 28px;
            font-weight: 600;
            margin: 0 0 10px 0;
            background: linear-gradient(135deg, #00ff87 0%%, #60efff 100%%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        p {
            color: #a0a5b5;
            font-size: 15px;
            line-height: 1.5;
            margin: 0 0 30px 0;
        }
        .qr-wrapper {
            background: #ffffff;
            padding: 24px;
            border-radius: 20px;
            display: inline-block;
            box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
            margin-bottom: 24px;
            transition: transform 0.3s ease;
        }
        .qr-wrapper:hover {
            transform: scale(1.02);
        }
        .qr-wrapper img {
            display: block;
            width: 240px;
            height: 240px;
        }
        .status-badge {
            font-size: 13px;
            font-weight: 600;
            padding: 8px 16px;
            border-radius: 30px;
            display: inline-block;
            background: rgba(0, 255, 135, 0.1);
            color: #00ff87;
            border: 1px solid rgba(0, 255, 135, 0.2);
            letter-spacing: 0.5px;
        }
        .footer {
            margin-top: 25px;
            font-size: 12px;
            color: #5d6275;
        }
    </style>
    <script>
        // Refresh QR code every 15 seconds
        setTimeout(() => {
            window.location.reload();
        }, 15000);
    </script>
</head>
<body>
    <div class="container">
        <h1>WhatsApp Link</h1>
        <p>Open WhatsApp on your phone, tap Menu or Settings &gt; Linked Devices, and scan this QR code.</p>
        <div class="qr-wrapper">
            <img src="https://api.qrserver.com/v1/create-qr-code/?size=240x240&margin=0&data=%s" alt="WhatsApp QR Code" />
        </div>
        <div>
            <span class="status-badge">AWAITING SCAN</span>
        </div>
        <div class="footer">
            QR refreshes automatically.
        </div>
    </div>
</body>
</html>`

const htmlAlreadyConnected = `<!DOCTYPE html>
<html>
<head>
    <title>WhatsApp Connected</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;600&display=swap" rel="stylesheet">
    <style>
        body {
            background: radial-gradient(circle at center, #1a1c24 0%%, #111218 100%%);
            color: #ffffff;
            font-family: 'Outfit', sans-serif;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            margin: 0;
            padding: 20px;
            box-sizing: border-box;
        }
        .container {
            background: rgba(255, 255, 255, 0.03);
            backdrop-filter: blur(20px);
            -webkit-backdrop-filter: blur(20px);
            border: 1px solid rgba(255, 255, 255, 0.08);
            padding: 40px;
            border-radius: 24px;
            box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5);
            text-align: center;
            max-width: 420px;
            width: 100%%;
            animation: fadeIn 0.8s ease-out;
        }
        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(20px); }
            to { opacity: 1; transform: translateY(0); }
        }
        .success-icon {
            font-size: 64px;
            margin-bottom: 20px;
            color: #00ff87;
            animation: scaleUp 0.5s cubic-bezier(0.175, 0.885, 0.32, 1.275);
        }
        @keyframes scaleUp {
            from { transform: scale(0); }
            to { transform: scale(1); }
        }
        h1 {
            font-size: 28px;
            font-weight: 600;
            margin: 0 0 10px 0;
            color: #ffffff;
        }
        p {
            color: #a0a5b5;
            font-size: 15px;
            line-height: 1.5;
            margin: 0 0 30px 0;
        }
        .status-badge {
            font-size: 13px;
            font-weight: 600;
            padding: 8px 16px;
            border-radius: 30px;
            display: inline-block;
            background: rgba(0, 255, 135, 0.1);
            color: #00ff87;
            border: 1px solid rgba(0, 255, 135, 0.2);
            letter-spacing: 0.5px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="success-icon">✓</div>
        <h1>Connected</h1>
        <p>Your WhatsApp session is active and ready to process requests.</p>
        <div>
            <span class="status-badge">ACTIVE SESSION</span>
        </div>
    </div>
</body>
</html>`

const htmlLoading = `<!DOCTYPE html>
<html>
<head>
    <title>WhatsApp Loading</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;600&display=swap" rel="stylesheet">
    <style>
        body {
            background: radial-gradient(circle at center, #1a1c24 0%%, #111218 100%%);
            color: #ffffff;
            font-family: 'Outfit', sans-serif;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            margin: 0;
            padding: 20px;
            box-sizing: border-box;
        }
        .container {
            background: rgba(255, 255, 255, 0.03);
            backdrop-filter: blur(20px);
            -webkit-backdrop-filter: blur(20px);
            border: 1px solid rgba(255, 255, 255, 0.08);
            padding: 40px;
            border-radius: 24px;
            box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5);
            text-align: center;
            max-width: 420px;
            width: 100%%;
        }
        .spinner {
            border: 4px solid rgba(255, 255, 255, 0.1);
            width: 50px;
            height: 50px;
            border-radius: 50%%;
            border-left-color: #60efff;
            animation: spin 1s linear infinite;
            display: inline-block;
            margin-bottom: 24px;
        }
        @keyframes spin {
            0%% { transform: rotate(0deg); }
            100%% { transform: rotate(360deg); }
        }
        h1 {
            font-size: 24px;
            font-weight: 600;
            margin: 0 0 10px 0;
            color: #ffffff;
        }
        p {
            color: #a0a5b5;
            font-size: 15px;
            margin: 0;
        }
    </style>
    <script>
        setTimeout(() => {
            window.location.reload();
        }, 3000);
    </script>
</head>
<body>
    <div class="container">
        <div class="spinner"></div>
        <h1>Please Wait</h1>
        <p>%s</p>
    </div>
</body>
</html>`
