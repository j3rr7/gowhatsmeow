## Base URL
By default, the server runs on:
`http://localhost:8080`

---

## 1. Health Check

Checks if the API server is online and running.

* **Endpoint**: `/healthz`
* **Method**: `GET`
* **Headers**: None
* **Request Body**: None

### Response
* **Code**: `200 OK`
* **Content-Type**: `application/json`
* **Response Body**:
```json
{
  "status": "healthy"
}
```

---

## 2. Get Status

Returns the current state of the WhatsApp client, its active JID, and the current QR code if available.

* **Endpoint**: `/api/v1/whatsapp/status`
* **Method**: `GET`
* **Headers**: None
* **Request Body**: None

### Responses
#### Success (Disconnected / QR Available)
* **Code**: `200 OK`
* **Response Body**:
```json
{
  "status": "qr_ready",
  "jid": "",
  "has_qr": true,
  "qr_code": "2@qP7x... (raw QR text payload)"
}
```

#### Success (Connected)
* **Code**: `200 OK`
* **Response Body**:
```json
{
  "status": "connected",
  "jid": "628123456789@s.whatsapp.net",
  "has_qr": false,
  "qr_code": ""
}
```

---

## 3. Get QR Code

Retrieves the current login QR code. If accessed directly from a web browser, it renders a sleek visual user interface for scanner pairing.

* **Endpoint**: `/api/v1/whatsapp/qr`
* **Method**: `GET`
* **Request Body**: None

### Responses
#### Browser View (HTML)
* **Trigger**: Headers contain `Accept: text/html`
* **Code**: `200 OK`
* **Content-Type**: `text/html`
* **Behavior**: Renders a dark-mode glassmorphic page displaying a visual QR code for scan. Refreshes automatically every 15s. If already logged in, displays a successful connection checkmark screen.

#### Client JSON (Not Connected, QR Available)
* **Trigger**: Headers contain `Accept: application/json`
* **Code**: `200 OK`
* **Content-Type**: `application/json`
* **Response Body**:
```json
{
  "status": "qr_ready",
  "qr_code": "2@qP7x..."
}
```

#### Client JSON (Already Connected)
* **Code**: `200 OK`
* **Response Body**:
```json
{
  "status": "connected",
  "message": "Already connected"
}
```

#### Client JSON (QR Not Yet Ready)
* **Code**: `202 Accepted`
* **Response Body**:
```json
{
  "status": "connecting",
  "message": "QR code is not generated yet, please try again in a few seconds"
}
```

---

## 4. Send Message (Text)

Dispatches a plain text message to a contact.

* **Endpoint**: `/api/v1/whatsapp/send`
* **Method**: `POST`
* **Headers**:
  * `Content-Type: application/json`
* **Request Body**:
```json
{
  "to": "628123456789",
  "message": "Hello world!"
}
```
*Note: Recipient phone numbers without `@s.whatsapp.net` suffix are automatically formatted.*

### Responses
#### Success
* **Code**: `200 OK`
* **Response Body**:
```json
{
  "success": true,
  "message_id": "ABC123XYZ"
}
```

#### Bad Request (Missing fields)
* **Code**: `400 Bad Request`
* **Response Body**:
```json
{
  "message": "Recipient ('to') is required"
}
```

#### Client Not Connected
* **Code**: `500 Internal Server Error`
* **Response Body**:
```json
{
  "message": "whatsapp client is not connected"
}
```

---

## 5. Send File (Media/Documents)

Uploads a document or media file to WhatsApp's servers and sends it to a recipient.

* **Endpoint**: `/api/v1/whatsapp/send-file`
* **Method**: `POST`
* **Headers**:
  * `Content-Type: multipart/form-data`
* **Request Body**: Use multipart form fields:
  * `to` (string, required): Phone number or JID of recipient.
  * `file` (file, required): The document, image, audio, or video file.
  * `caption` (string, optional): Text description below the file (supported for images/videos).

### Media Detection Logic
The API checks the file's mime-type header to format the message properties:
- `image/*` -> Sent as image message (supports captions).
- `audio/*` -> Sent as audio message.
- `video/*` -> Sent as video message (supports captions).
- Any other -> Sent as standard document attachment.

### Responses
#### Success
* **Code**: `200 OK`
* **Response Body**:
```json
{
  "success": true,
  "message_id": "MEDIA123"
}
```

#### Bad Request (Missing file or recipient)
* **Code**: `400 Bad Request`
* **Response Body**:
```json
{
  "message": "File ('file') is required"
}
```
---

## 6. Logout

Disconnects from WhatsApp, revokes credentials, and clears database session stores.

* **Endpoint**: `/api/v1/whatsapp/logout`
* **Method**: `POST`
* **Headers**: None
* **Request Body**: None

### Responses
#### Success
* **Code**: `200 OK`
* **Response Body**:
```json
{
  "success": true,
  "message": "Logged out and session cleared successfully"
}
```

---

## 7. Check Number

Checks if a phone number or a list of phone numbers are registered on WhatsApp.

* **Endpoint**: `/api/v1/whatsapp/check-number`
* **Method**: `POST`
* **Headers**:
  * `Content-Type: application/json`
* **Request Body** (Single check):
```json
{
  "number": "+62 812-3456-789"
}
```
* **Request Body** (Bulk check):
```json
{
  "numbers": ["628123456789", "628999999999"]
}
```

### Responses
#### Success (Single check)
* **Code**: `200 OK`
* **Response Body**:
```json
{
  "registered": true,
  "number": "628123456789",
  "jid": "628123456789@s.whatsapp.net",
  "results": [
    {
      "number": "628123456789",
      "registered": true,
      "jid": "628123456789@s.whatsapp.net"
    }
  ]
}
```

#### Success (Bulk check)
* **Code**: `200 OK`
* **Response Body**:
```json
{
  "results": [
    {
      "number": "628123456789",
      "registered": true,
      "jid": "628123456789@s.whatsapp.net"
    },
    {
      "number": "628999999999",
      "registered": false,
      "jid": ""
    }
  ]
}
```

---

## 8. Get Webhook

Retrieves the currently registered webhook callback URL.

* **Endpoint**: `/api/v1/whatsapp/webhook`
* **Method**: `GET`
* **Headers**: None
* **Request Body**: None

### Responses
#### Success
* **Code**: `200 OK`
* **Response Body**:
```json
{
  "url": "https://your-app.com/api/v1/webhook"
}
```

---

## 9. Set Webhook

Registers or updates the webhook callback URL dynamically at runtime.

* **Endpoint**: `/api/v1/whatsapp/webhook`
* **Method**: `POST`
* **Headers**:
  * `Content-Type: application/json`
* **Request Body**:
```json
{
  "url": "https://your-app.com/api/v1/webhook"
}
```

### Responses
#### Success
* **Code**: `200 OK`
* **Response Body**:
```json
{
  "success": true,
  "message": "Webhook URL updated successfully",
  "url": "https://your-app.com/api/v1/webhook"
}
```

