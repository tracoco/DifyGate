# WhatsApp Graph API Integration with DifyGate

This document provides example curl commands to test the WhatsApp integration capabilities of DifyGate with the Meta Graph API.

## Prerequisites

Before using these commands, you'll need:

1. A WhatsApp Business Account
2. Access to the WhatsApp Business API
3. Your WhatsApp Business Phone Number ID
4. A Meta Graph API Token
5. DifyGate properly configured with your WhatsApp credentials

## Environment Setup

To make the curl commands easier to use, set these environment variables:

```bash
# Your WhatsApp Business Phone Number ID
export DIFYGATE_PHONE_ID="your_whatsapp_phone_number_id"

# Your Permanent Graph API Token
export DIFYGATE_GRAPH_API_TOKEN="your_DIFYGATE_GRAPH_API_TOKEN"

# DifyGate API Key
export DIFYGATE_API_KEY="your_difygate_api_key"

# DifyGate Base URL
export DIFYGATE_URL="https://your-difygate-deployment.vercel.app"
```

## 1. Testing the WhatsApp Webhook

### Verifying the Webhook URL

Meta verifies your webhook URL with a GET request. You can test this verification with:

```bash
curl -X GET "${DIFYGATE_URL}/api/v1/whatsapp/webhook?hub.mode=subscribe&hub.verify_token=YOUR_WEBHOOK_VERIFY_TOKEN&hub.challenge=challenge_string"
```

A successful verification should return the challenge string.

### Simulating a WhatsApp Message Webhook

Send a test POST request to simulate receiving a WhatsApp message:

```bash
curl -X POST "${DIFYGATE_URL}/api/v1/whatsapp/webhook" \
  -H "Content-Type: application/json" \
  -d '{
    "entry": [{
      "changes": [{
        "value": {
          "metadata": {
            "phone_number_id": "'${DIFYGATE_PHONE_ID}'"
          },
          "messages": [{
            "from": "recipient_phone_number",
            "id": "test_message_id",
            "text": {
              "body": "Hello from curl test"
            },
            "type": "text"
          }]
        }
      }]
    }]
  }'
```

## 2. Direct Graph API Interactions

Refer to https://developers.facebook.com/docs/whatsapp/cloud-api/messages/text-messages for more details.

### Sending a Text Message

Send a simple text message to a WhatsApp user (the recipient_phone_number need to send an init message to the account, otherwise WhatsApp need to init chat from a template message):

```bash
curl -X POST "https://graph.facebook.com/v22.0/${DIFYGATE_PHONE_ID}/messages" \
  -H "Authorization: Bearer ${DIFYGATE_GRAPH_API_TOKEN}" \
  -H "Content-Type: application/json" \
-d '
{
  "messaging_product": "whatsapp",
  "recipient_type": "individual",
  "to": "recipient_phone_number",
  "type": "text",
  "text": {
    "preview_url": true,
    "body": "As requested, here'\''s the link to our latest product: https://www.meta.com/quest/quest-3/"
  }
}'
```

### Sending a Template Message

Send a predefined template message:

```bash
curl -X POST "https://graph.facebook.com/v22.0/${DIFYGATE_PHONE_ID}/messages" \
  -H "Authorization: Bearer ${DIFYGATE_GRAPH_API_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "messaging_product": "whatsapp",
    "to": "recipient_phone_number",
    "type": "template",
    "template": {
      "name": "hello_world",
      "language": {
        "code": "en_US"
      }
    }
  }'
```

## 3. Testing DifyGate's API Endpoints

### Health Check

Test if DifyGate is running correctly:

```bash
curl -X GET "${DIFYGATE_URL}/api/v1/health" \
  -H "Authorization: Bearer ${DIFYGATE_API_KEY}"
```

### Sending an Email

Test the email sending functionality:

```bash
curl -X POST "${DIFYGATE_URL}/api/v1/emails/send" \
  -H "Authorization: Bearer ${DIFYGATE_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "recipient@example.com",
    "subject": "Test Email from DifyGate",
    "body": "This is a test email sent via the DifyGate API.",
    "html": false
  }'
```

## 4. Debugging WhatsApp Integration

### Getting WhatsApp Business Account Information

```bash
curl -X GET "https://graph.facebook.com/v22.0/me/whatsapp_business_account" \
  -H "Authorization: Bearer ${DIFYGATE_GRAPH_API_TOKEN}"
```

### Getting Phone Numbers

List all phone numbers associated with your WhatsApp Business Account:

```bash
curl -X GET "https://graph.facebook.com/v22.0/me/phone_numbers" \
  -H "Authorization: Bearer ${DIFYGATE_GRAPH_API_TOKEN}"
```

### Checking Message Status

Check the status of a sent message:

```bash
curl -X GET "https://graph.facebook.com/v22.0/${MESSAGE_ID}" \
  -H "Authorization: Bearer ${DIFYGATE_GRAPH_API_TOKEN}"
```

## Troubleshooting

### Common Issues and Solutions

1. **401 Unauthorized Error**:
   - Verify your Graph API token is correct and hasn't expired
   - Make sure the token has the right permissions

2. **400 Bad Request Error**:
   - Check your payload formatting
   - Ensure phone numbers are in the correct format with country code

3. **Webhook Not Receiving Messages**:
   - Verify your webhook URL is correctly configured in Meta Developer Dashboard
   - Check that your webhook verification token matches in both Meta Dashboard and DifyGate

4. **Message Delivery Failures**:
   - Ensure the recipient phone number is a valid WhatsApp user
   - Check if the recipient has opted in to receive messages

### Checking API Rate Limits

```bash
curl -X GET "https://graph.facebook.com/v22.0/app/rate_limit_usage" \
  -H "Authorization: Bearer ${DIFYGATE_GRAPH_API_TOKEN}"
```

## Best Practices

1. Always use environment variables for sensitive information like tokens
2. Test message templates in the Meta Business Manager before using them in API calls
3. Implement proper error handling in your production code
4. Monitor your API usage to avoid hitting rate limits
5. Use the WhatsApp Business API for business communications only, following WhatsApp's policies

## Additional Resources

- [Meta Graph API Documentation](https://developers.facebook.com/docs/whatsapp/api/messages/)
- [WhatsApp Business Platform](https://developers.facebook.com/docs/whatsapp/overview)
- [Meta Developer Dashboard](https://developers.facebook.com/apps/)

## Setting Up Webhooks in Meta Developer Dashboard

To receive WhatsApp messages through DifyGate, you need to configure webhooks in the Meta Developer Dashboard:

1. Go to [Meta Developer Dashboard](https://developers.facebook.com/apps/)
2. Select your app
3. Go to WhatsApp > Configurations
4. Under Webhooks, click "Configure"
5. Enter your DifyGate webhook URL: `https://your-difygate-deployment.vercel.app/api/v1/whatsapp/webhook`
6. Enter the verification token you've set as `DIFYGATE_WEBHOOK_VERIFY_TOKEN` in DifyGate
7. Select the subscription fields: `messages`, `message_deliveries`, `messaging_postbacks`, etc.
8. Click "Verify and Save"

## Customizing Message Responses in DifyGate

DifyGate can be customized to process incoming WhatsApp messages and send tailored responses. Edit the `HandleWhatsAppWebhookPost` function in the `wa_webhook.go` file to implement custom behavior:

```go
// Example custom response logic
if strings.Contains(strings.ToLower(message.Text.Body), "help") {
    // Send a help message
    sendCustomMessage(businessPhoneNumberID, message.From, "Available commands: help, info, status")
} else if strings.Contains(strings.ToLower(message.Text.Body), "info") {
    // Send information
    sendCustomMessage(businessPhoneNumberID, message.From, "DifyGate is a flexible API gateway service.")
} else {
    // Default echo response
    sendReplyMessage(businessPhoneNumberID, message.From, message.Text.Body, message.ID)
}
```

## Security Considerations

1. **API Key Protection**: Always protect your Graph API token and never expose it in client-side code
2. **Webhook Verification**: Implement proper verification for your webhook to prevent spoofed messages
3. **Input Validation**: Validate all incoming message content before processing
4. **Rate Limiting**: Implement rate limiting to prevent abuse
5. **Error Handling**: Properly handle and log errors for troubleshooting