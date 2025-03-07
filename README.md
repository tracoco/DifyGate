# DifyGate

A RESTful email service API built with Go.

## Features

- Send emails with plain text or HTML content
- Support for CC and BCC recipients
- Support for file attachments
- JSON API with proper error handling
- Configurable SMTP settings via environment variables

## Getting Started

### Prerequisites

- Go 1.18 or newer
- SMTP server access (e.g., Gmail)

### Configuration

Create a `.env` file in the project root with the following variables:

```
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM_NAME=DifyGate Email Service
```

For Gmail, you'll need to create an "App Password" in your Google Account security settings.

### Running the Server

```bash
go mod tidy
go run main.go
```

The server will start on port 6001.

## API Endpoints

### Send Email

```
#POST /api/v1/emails/send
curl -X POST http://localhost:6001/api/v1/emails/send \
-H "Content-Type: application/json" \
-d '{
  "to": ["recipient@example.com"],
  "subject": "Hello from DifyGate",
  "body": "This is a test email from DifyGate API"
}'
```

Request body:

```json
{
  "to": ["recipient@example.com"],
  "cc": ["cc-recipient@example.com"],
  "bcc": ["bcc-recipient@example.com"],
  "subject": "Hello from DifyGate",
  "body": "This is a test email from DifyGate API",
  "is_html": false,
  "attachments": [
    {
      "filename": "test.txt",
      "data": "SGVsbG8gV29ybGQh",
      "mime_type": "text/plain"
    }
  ]
}
```

- `to`: Array of recipient email addresses (required)
- `cc`: Array of CC recipient email addresses (optional)
- `bcc`: Array of BCC recipient email addresses (optional)
- `subject`: Email subject (required)
- `body`: Email body content (required)
- `is_html`: Set to true if the body is HTML content (default: false)
- `attachments`: Array of file attachments (optional)
  - `filename`: Name of the file
  - `data`: Base64-encoded file content
  - `mime_type`: MIME type of the file

Response:

```json
{
  "message": "Email sent successfully"
}
```

### Health Check

```
# GET /api/v1/health
curl http://localhost:6001/api/v1/health
```

Response:

```json
{
  "status": "ok",
  "service": "DifyGate",
  "timestamp": "2025-03-06T12:34:56Z"
}
```
