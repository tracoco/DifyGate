# DifyGate

DifyGate is a flexible API gateway service that handles WhatsApp message webhooks and email sending capabilities.

## Deploying to Vercel

DifyGate can be easily deployed to Vercel as a serverless application. Follow these steps for deployment:

### Prerequisites

1. A [Vercel](https://vercel.com) account
2. [Vercel CLI](https://vercel.com/download) installed (optional for local development)
3. Required environment variables (see below)

### Environment Variables

Set the following environment variables in your Vercel project settings:

#### Required Variables
- `DIFYGATE_API_KEY`: Secret key for authenticating API requests
- `DIFYGATE_SMTP_HOST`: SMTP server host (e.g., smtp.gmail.com)
- `DIFYGATE_SMTP_PORT`: SMTP server port (e.g., 587)
- `DIFYGATE_SMTP_USERNAME`: SMTP username/email
- `DIFYGATE_SMTP_PASSWORD`: SMTP password or app password

#### WhatsApp Integration Variables
- `DIFYGATE_WEBHOOK_VERIFY_TOKEN`: Verification token for WhatsApp webhook
- `DIFYGATE_GRAPH_API_TOKEN`: Meta Graph API token for WhatsApp Business API

### Deployment Steps

1. Clone the repository:
   ```bash
   git clone https://github.com/tracoco/DifyGate.git
   cd DifyGate
   ```

2. Link to your Vercel project:
   ```bash
   vercel link
   ```

3. Add environment variables:
   ```bash
   vercel env add DIFYGATE_API_KEY
   vercel env add DIFYGATE_SMTP_HOST
   # Add other required variables...
   ```

4. Deploy to Vercel:
   ```bash
   vercel --prod
   ```

### Alternative Deployment Method

You can also connect your GitHub repository to Vercel for automatic deployments:

1. Push your code to GitHub
2. Import the project in the Vercel dashboard
3. Configure environment variables in the Vercel project settings
4. Deploy the project

## API Documentation

### Authentication

All API endpoints (except WhatsApp webhooks) require a Bearer token for authentication:

```
Authorization: Bearer YOUR_DIFYGATE_API_KEY
```

### WhatsApp Webhook

- `GET /api/v1/whatsapp/webhook`: Used by Meta for webhook verification
- `POST /api/v1/whatsapp/webhook`: Receives WhatsApp messages

### Email Service

- `POST /api/v1/emails/send`: Sends emails through the configured SMTP server

Example request:
```json
{
  "to": "recipient@example.com",
  "subject": "Hello from DifyGate",
  "body": "This is a test email."
}
```

### Health Check

- `GET /api/v1/health`: Checks if the service is running correctly

## Local Development

To run the service locally:

```bash
# Set required environment variables
export DIFYGATE_API_KEY=your_api_key
export DIFYGATE_SMTP_HOST=smtp.example.com
# Set other required variables...

# Run the service
go run main.go
```

The server will start on port 6001 by default.