package gateapi

import (
	"bytes"
	"context"
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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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

// WhatsAppHandler manages WhatsApp webhook handling
type WhatsAppHandler struct {
	log         *logrus.Logger
	difyHandler *DifyHandler
}

// NewWhatsAppHandler creates a new WhatsApp webhook handler
func NewWhatsAppHandler(log *logrus.Logger) *WhatsAppHandler {
	return &WhatsAppHandler{
		log:         log,
		difyHandler: NewDifyHandler(log),
	}
}

// HandleWhatsAppWebhookPost handles POST requests to the WhatsApp webhook
func (h *WhatsAppHandler) HandleWhatsAppWebhookPost(c *gin.Context) {
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
		h.log.WithField("message", string(body)).Info("Incoming webhook message")
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

			// Process the message asynchronously
			// We don't want to block the webhook response
			go h.processWhatsAppMessage(businessPhoneNumberID, message.From, message.Text.Body, message.ID)

			// Mark incoming message as read
			markMessageAsRead(businessPhoneNumberID, message.ID)
		}
	}

	// Return 200 OK (must respond quickly to webhook)
	c.Status(http.StatusOK)
}

// processWhatsAppMessage handles the WhatsApp message processing and Dify integration
func (h *WhatsAppHandler) processWhatsAppMessage(phoneNumberID, from, messageBody, messageID string) {
	// Send initial acknowledgment
	initialResponse := "I'm processing your request..."
	sendReplyMessage(phoneNumberID, from, initialResponse, messageID)

	// Create context with reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Use user's WhatsApp number as the conversation ID to maintain context
	// Format the phone number to ensure it's consistent
	userID := strings.TrimPrefix(from, "+")

	// Prepare request to Dify
	difyReq := DifyChatMessageRequest{
		Inputs:         map[string]interface{}{},
		Query:          messageBody,
		User:           userID,      // Set the user ID as the WhatsApp number
		ConversationID: "",          //"whatsapp_" + userID, // Prefix to ensure uniqueness
		ResponseMode:   "streaming", // Use streaming for real-time responses
	}

	// Start streaming response from Dify
	h.log.WithFields(logrus.Fields{
		"userID": userID,
		"query":  messageBody,
	}).Info("Sending request to Dify")

	respChan, errChan := h.difyHandler.DifyChatMessageStreaming(ctx, difyReq)

	// Variables to build the complete response
	var fullAnswer strings.Builder
	var lastMessageSent time.Time
	var lastChunkSize int
	const minSendInterval = 1500 * time.Millisecond // Minimum 1.5 seconds between messages
	const minChunkSize = 50                         // Minimum characters per chunk

	// Process streaming responses
	for {
		select {
		case err, ok := <-errChan:
			if !ok {
				// Error channel closed, no errors occurred
				continue
			}

			// Something went wrong
			h.log.WithError(err).Error("Error in Dify streaming response")
			errorMessage := fmt.Sprintf("Sorry, I encountered an error: %s", err.Error())
			sendReplyMessage(phoneNumberID, from, errorMessage, messageID)
			return

		case resp, ok := <-respChan:
			h.log.WithField("response", resp).Info("Received Dify response")
			if !ok {
				// Response channel closed, stream completed
				// Send any remaining text
				if fullAnswer.Len() > 0 {
					sendReplyMessage(phoneNumberID, from, fullAnswer.String(), messageID)
				}
				return
			}

			// Only process message chunks from the "message_delta" event
			if resp.Event == "message_delta" && resp.Answer != "" {
				// Accumulate the answer
				fullAnswer.WriteString(resp.Answer)

				// Determine if we should send a partial response
				timeSinceLastMessage := time.Since(lastMessageSent)
				currentSize := fullAnswer.Len()

				// Send partial response if enough time has passed and we have enough new content
				if timeSinceLastMessage > minSendInterval && (currentSize-lastChunkSize) >= minChunkSize {
					partialResponse := fullAnswer.String()
					sendReplyMessage(phoneNumberID, from, partialResponse, messageID)

					// Reset buffer after sending
					fullAnswer.Reset()
					lastChunkSize = 0
					lastMessageSent = time.Now()
				}
			} else if resp.Event == "message_end" {
				// Message is complete, send any remaining text
				if fullAnswer.Len() > 0 {
					finalResponse := fullAnswer.String()
					sendReplyMessage(phoneNumberID, from, finalResponse, messageID)
				}
			}

		case <-ctx.Done():
			// Context timeout or cancellation
			h.log.Warn("Context canceled or timed out while processing Dify response")
			timeoutMessage := "Sorry, the response took too long. Please try again later."
			sendReplyMessage(phoneNumberID, from, timeoutMessage, messageID)
			return
		}
	}
}

// HandleWhatsAppWebhookGet handles GET requests to the WhatsApp webhook (for verification)
func (h *WhatsAppHandler) HandleWhatsAppWebhookGet(c *gin.Context) {
	webhookVerifyToken := os.Getenv("DIFYGATE_WEBHOOK_VERIFY_TOKEN")

	// Get query parameters
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	// Check the mode and token sent are correct
	if mode == "subscribe" && token == webhookVerifyToken {
		// Respond with 200 OK and challenge token from the request
		c.String(http.StatusOK, challenge)
		h.log.Info("Webhook verified successfully!")
	} else {
		// Respond with '403 Forbidden' if verify tokens do not match
		c.Status(http.StatusForbidden)
		h.log.Warn("Webhook verification failed")
	}
}

func sendReplyMessage(phoneNumberID, to, messageBody, messageID string) {
	graphAPIToken := os.Getenv("DIFYGATE_GRAPH_API_TOKEN")
	url := fmt.Sprintf("https://graph.facebook.com/v22.0/%s/messages", phoneNumberID)

	// Truncate message if too long for WhatsApp (limit is around 4096 characters)
	if len(messageBody) > 4000 {
		messageBody = messageBody[:3997] + "..."
	}

	// Create request payload
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"text": map[string]string{
			"body": messageBody,
		},
		"context": map[string]string{
			"message_id": messageID,
		},
	}

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

	// Log response for debugging
	if os.Getenv("DIFYGATE_DEBUG") == "true" {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("WhatsApp API response: %s", respBody)
	}
}

func markMessageAsRead(phoneNumberID, messageID string) {
	graphAPIToken := os.Getenv("DIFYGATE_GRAPH_API_TOKEN")
	url := fmt.Sprintf("https://graph.facebook.com/v22.0/%s/messages", phoneNumberID)

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
