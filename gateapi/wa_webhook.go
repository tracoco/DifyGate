package gateapi

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// WebhookRequest represents the incoming WhatsApp webhook payload
type WebhookRequest struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				Metadata struct {
					PhoneNumberID string `json:"phone_number_id"`
				} `json:"metadata"`
				Messages []struct {
					From string `json:"from"`
					ID   string `json:"id"`
					Text struct {
						Body string `json:"body"`
					} `json:"text"`
					Type string `json:"type"`
				} `json:"messages"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

// VerifyWebhook verifies the authenticity of the webhook request by comparing HMAC signatures
func VerifyWebhook(data []byte, hmacHeader string) bool {
	// Remove prefix if present
	hmacReceived := hmacHeader
	if strings.HasPrefix(hmacReceived, "sha256=") {
		hmacReceived = strings.TrimPrefix(hmacReceived, "sha256=")
	}

	// Get the API secret from environment variables
	appSecret := os.Getenv("DIFYGATE_WHATSAPP_APP_SECRET")

	// Create HMAC hash using SHA-256
	h := hmac.New(sha256.New, []byte(appSecret))
	h.Write(data)
	digest := hex.EncodeToString(h.Sum(nil))

	// Compare the calculated digest with the received one
	// This is a constant-time comparison to prevent timing attacks
	return hmac.Equal([]byte(hmacReceived), []byte(digest))
}

// logRequestHeaders prints all headers from the request
func logRequestHeaders(c *gin.Context) {
	if os.Getenv("DIFYGATE_DEBUG") != "true" {
		return
	}

	log.Println("--- Request Headers ---")

	// Get all headers
	headers := c.Request.Header

	// Print each header
	for name, values := range headers {
		for _, value := range values {
			log.Printf("%s: %s\n", name, value)
		}
	}

	// Specifically check for the signature header that we care about
	sigHeader := c.GetHeader("X-Hub-Signature-256")
	log.Printf("X-Hub-Signature-256: %s\n", sigHeader)

	log.Println("----------------------")
}

// HandleWhatsAppWebhookPost handles POST requests to the WhatsApp webhook
func HandleWhatsAppWebhookPost(c *gin.Context) {
	logRequestHeaders(c)
	// Read the request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	if !VerifyWebhook(body, c.GetHeader("X-Hub-Signature-256")) {
		// Respond with '403 Forbidden' if verify signature do not match
		c.Status(http.StatusForbidden)
		return
	}

	// Log incoming messages
	if os.Getenv("DIFYGATE_DEBUG") == "true" {
		log.Printf("Incoming webhook message: %s\n", string(body))
	}

	// Parse the request body
	var webhookRequest WebhookRequest
	if err := json.Unmarshal(body, &webhookRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request body"})
		return
	}

	// Check if the webhook request contains a message
	if len(webhookRequest.Entry) > 0 && len(webhookRequest.Entry[0].Changes) > 0 &&
		len(webhookRequest.Entry[0].Changes[0].Value.Messages) > 0 {

		message := webhookRequest.Entry[0].Changes[0].Value.Messages[0]

		// Check if the incoming message contains text
		if message.Type == "text" {
			// Extract the business number to send the reply from it
			businessPhoneNumberID := webhookRequest.Entry[0].Changes[0].Value.Metadata.PhoneNumberID

			// Send a reply message
			sendReplyMessage(businessPhoneNumberID, message.From, message.Text.Body, message.ID)

			// Mark incoming message as read
			markMessageAsRead(businessPhoneNumberID, message.ID)
		}
	}

	// Return 200 OK
	c.Status(http.StatusOK)
}

// HandleWhatsAppWebhookGet handles GET requests to the WhatsApp webhook (for verification)
func HandleWhatsAppWebhookGet(c *gin.Context) {
	webhookVerifyToken := os.Getenv("DIFYGATE_WEBHOOK_VERIFY_TOKEN")

	// Get query parameters
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	// Check the mode and token sent are correct
	if mode == "subscribe" && token == webhookVerifyToken {
		// Respond with 200 OK and challenge token from the request
		c.String(http.StatusOK, challenge)
		log.Println("Webhook verified successfully!")
	} else {
		// Respond with '403 Forbidden' if verify tokens do not match
		c.Status(http.StatusForbidden)
	}
}

func sendReplyMessage(phoneNumberID, to, messageBody, messageID string) {
	graphAPIToken := os.Getenv("DIFYGATE_GRAPH_API_TOKEN")
	url := fmt.Sprintf("https://graph.facebook.com/v22.0/%s/messages", phoneNumberID)

	// Create request payload
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"text": map[string]string{
			"body": "Echo: " + messageBody,
		},
		"context": map[string]string{
			"message_id": messageID,
		},
	}

	log.Println("Sending reply message:", payload)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal reply payload: %v", err)
		return
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("Failed to create reply request: %v", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+graphAPIToken)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send reply: %v", err)
		return
	}
	defer resp.Body.Close()
}

func markMessageAsRead(phoneNumberID, messageID string) {
	graphAPIToken := os.Getenv("GRAPH_API_TOKEN")
	url := fmt.Sprintf("https://graph.facebook.com/v18.0/%s/messages", phoneNumberID)

	// Create request payload
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"status":            "read",
		"message_id":        messageID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal read status payload: %v", err)
		return
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("Failed to create read status request: %v", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+graphAPIToken)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to mark message as read: %v", err)
		return
	}
	defer resp.Body.Close()
}
