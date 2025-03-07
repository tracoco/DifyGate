package gateapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// DifyHandler handles Dify API integration
type DifyHandler struct {
	log          *logrus.Logger
	difyBaseURL  string
	difyAPIKey   string
	difyClientID string
}

// NewDifyHandler creates a new Dify API handler
func NewDifyHandler(log *logrus.Logger) *DifyHandler {
	return &DifyHandler{
		log:          log,
		difyBaseURL:  getEnvOrDefault("DIFYGATE_DIFY_BASE_URL", "https://api.dify.ai/v1"),
		difyAPIKey:   getEnvOrDefault("DIFYGATE_DIFY_API_KEY", ""),
		difyClientID: getEnvOrDefault("DIFYGATE_DIFY_CLIENT_ID", ""),
	}
}

// Helper function to get environment variable with default value
func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// ChatMessageRequest represents the request body for the Dify chat-message API
type ChatMessageRequest struct {
	Query          string                 `json:"query"`
	ResponseMode   string                 `json:"response_mode,omitempty"` // blocking or streaming
	User           string                 `json:"user,omitempty"`
	Inputs         map[string]interface{} `json:"inputs"`
	Files          []string               `json:"files,omitempty"` // Array of file IDs
	ConversationID string                 `json:"conversation_id,omitempty"`
}

// ChatMessageResponse represents the response from the Dify chat-message API
type ChatMessageResponse struct {
	ID                   string                 `json:"id"`
	Answer               string                 `json:"answer"`
	ConversationID       string                 `json:"conversation_id"`
	CreatedAt            int64                  `json:"created_at"`
	InvokedAgent         interface{}            `json:"invoked_agent"`
	MetaSuggestions      []string               `json:"meta_suggestions,omitempty"`
	UserInputs           map[string]interface{} `json:"user_inputs,omitempty"`
	TextResponses        []TextResponse         `json:"text_responses,omitempty"`
	AgentThought         AgentThought           `json:"agent_thought,omitempty"`
	ReturnToUserMessages []interface{}          `json:"return_to_user_messages,omitempty"`
}

// StreamingChatResponse represents a streaming response chunk from Dify
type StreamingChatResponse struct {
	Event        string      `json:"event"`
	ID           string      `json:"id,omitempty"`
	Answer       string      `json:"answer,omitempty"`
	Metadata     interface{} `json:"metadata,omitempty"`
	ErrorMsg     string      `json:"error,omitempty"`
	Status       string      `json:"status,omitempty"`
	FinishReason string      `json:"finish_reason,omitempty"`
}

// TextResponse represents a text response segment from Dify
type TextResponse struct {
	Text string `json:"text"`
}

// AgentThought represents the agent thought process data
type AgentThought struct {
	AgentName   string      `json:"agent_name"`
	Thought     string      `json:"thought"`
	Tool        string      `json:"tool"`
	ToolInput   interface{} `json:"tool_input"`
	Observation string      `json:"observation"`
}

// DifyChatMessageRequest is the request format for the DifyChatMessage function
type DifyChatMessageRequest struct {
	Query          string                 `json:"query" binding:"required"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	User           string                 `json:"user,omitempty"`
	Inputs         map[string]interface{} `json:"inputs,omitempty"`
	ResponseMode   string                 `json:"response_mode,omitempty"`
}

// DifyChatMessage sends a message to Dify API and returns the response
func (h *DifyHandler) DifyChatMessage(req DifyChatMessageRequest) (*ChatMessageResponse, error) {
	// Prepare request to Dify API
	difyReq := ChatMessageRequest{
		Query:          req.Query,
		User:           req.User,
		Inputs:         req.Inputs,
		ConversationID: req.ConversationID,
		ResponseMode:   req.ResponseMode,
	}

	// Default to blocking if not specified
	if difyReq.ResponseMode == "" {
		difyReq.ResponseMode = "blocking"
	}

	// For streaming mode, we should use the streaming handler
	if difyReq.ResponseMode == "streaming" {
		return nil, fmt.Errorf("streaming mode not supported in DifyChatMessage, use HandleDifyChatMessageStreaming instead")
	}

	// Convert request to JSON
	reqBody, err := json.Marshal(difyReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to marshal Dify request")
		return nil, fmt.Errorf("failed to prepare request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/chat-messages", h.difyBaseURL)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		h.log.WithError(err).Error("Failed to create HTTP request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	if h.difyAPIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+h.difyAPIKey)
	}
	if h.difyClientID != "" {
		httpReq.Header.Set("X-Client-Id", h.difyClientID)
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to send request to Dify API")
		return nil, fmt.Errorf("failed to communicate with Dify API: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		h.log.WithError(err).Error("Failed to read Dify API response")
		return nil, fmt.Errorf("failed to read API response: %w", err)
	}

	// Check if response is successful
	if resp.StatusCode != http.StatusOK {
		h.log.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"response":    string(respBody),
		}).Error("Dify API returned error")
		return nil, fmt.Errorf("Dify API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Parse Dify response
	var difyResp ChatMessageResponse
	if err := json.Unmarshal(respBody, &difyResp); err != nil {
		h.log.WithError(err).Error("Failed to parse Dify API response")
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	return &difyResp, nil
}

// DifyChatMessageStreaming sends a message to Dify API and returns the response as a stream
func (h *DifyHandler) DifyChatMessageStreaming(ctx context.Context, req DifyChatMessageRequest) (chan StreamingChatResponse, chan error) {
	// Initialize channels for the stream
	responseChan := make(chan StreamingChatResponse, 100) // Buffer to prevent blocking
	errChan := make(chan error, 1)                        // Buffered to avoid blocking

	// Enforce streaming mode
	req.ResponseMode = "streaming"

	// Start processing in a goroutine
	go func() {
		defer close(responseChan)
		defer close(errChan)

		// Prepare request to Dify API
		difyReq := ChatMessageRequest{
			Query:          req.Query,
			User:           req.User,
			Inputs:         req.Inputs,
			ConversationID: req.ConversationID,
			ResponseMode:   "streaming",
		}

		// Log beautified request for debugging
		if os.Getenv("DIFYGATE_DEBUG") == "true" {
			prettyJSON, err := json.MarshalIndent(difyReq, "", "  ")
			if err == nil {
				h.log.WithField("dify_request", string(prettyJSON)).Info("Dify streaming request")
			}
		}

		// Convert request to JSON
		reqBody, err := json.Marshal(difyReq)
		if err != nil {
			h.log.WithError(err).Error("Failed to marshal Dify streaming request")
			errChan <- fmt.Errorf("failed to prepare streaming request: %w", err)
			return
		}

		// Create HTTP request
		url := fmt.Sprintf("%s/chat-messages", h.difyBaseURL)
		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
		if err != nil {
			h.log.WithError(err).Error("Failed to create HTTP streaming request")
			errChan <- fmt.Errorf("failed to create streaming request: %w", err)
			return
		}

		// Set headers
		httpReq.Header.Set("Content-Type", "application/json")
		//httpReq.Header.Set("Accept", "text/event-stream")
		if h.difyAPIKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+h.difyAPIKey)
		}
		/* 		if h.difyClientID != "" {
			httpReq.Header.Set("X-Client-Id", h.difyClientID)
		} */

		// Log detailed request info
		h.log.WithFields(logrus.Fields{
			"url":    url,
			"method": "POST",
		}).Info("Sending streaming request to Dify API")

		// Send request
		client := &http.Client{
			Timeout: 0, // No timeout for streaming requests
		}
		resp, err := client.Do(httpReq)
		if err != nil {
			h.log.WithError(err).Error("Failed to send streaming request to Dify API")
			errChan <- fmt.Errorf("failed to communicate with Dify API: %w", err)
			return
		}
		defer resp.Body.Close()

		// Check response status
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			h.log.WithFields(logrus.Fields{
				"status_code": resp.StatusCode,
				"response":    string(body),
			}).Error("Dify API returned error for streaming request")
			errChan <- fmt.Errorf("Dify API streaming error (status %d): %s", resp.StatusCode, string(body))
			return
		}

		// Log that we're starting to process the stream
		h.log.Info("Starting to process Dify SSE stream")

		// Process the SSE stream
		scanner := bufio.NewScanner(resp.Body)
		var eventData []byte

		for scanner.Scan() {
			line := scanner.Text()

			// Debug each line received in the SSE stream
			if os.Getenv("DIFYGATE_DEBUG") == "true" {
				h.log.WithField("sse_line", line).Debug("Received SSE line")
			}

			// Empty line signals the end of an event
			if line == "" {
				if len(eventData) > 0 {
					// Process complete event
					processEvent(eventData, responseChan, h.log)
					eventData = nil
				}
				continue
			}

			// Lines starting with "data:" contain the event data
			if strings.HasPrefix(line, "data:") {
				data := strings.TrimPrefix(line, "data:")
				data = strings.TrimSpace(data)
				eventData = append(eventData, []byte(data)...)
			}

			// Check context cancellation
			select {
			case <-ctx.Done():
				h.log.Info("Context canceled, stopping SSE processing")
				return
			default:
				// Continue processing
			}
		}

		// Check for scanner errors
		if err := scanner.Err(); err != nil {
			if err != io.EOF && !strings.Contains(err.Error(), "context canceled") {
				h.log.WithError(err).Error("Error reading SSE stream")
				errChan <- fmt.Errorf("error reading SSE stream: %w", err)
			} else {
				h.log.Info("SSE stream ended")
			}
		}
	}()

	return responseChan, errChan
}

// Helper function to process SSE events
func processEvent(data []byte, responseChan chan StreamingChatResponse, log *logrus.Logger) {
	// Skip empty data
	if len(data) == 0 || string(data) == "" {
		return
	}

	// Debug the raw data
	if os.Getenv("DIFYGATE_DEBUG") == "true" {
		log.WithField("event_data", string(data)).Debug("Processing SSE event data")
	}

	var response StreamingChatResponse
	if err := json.Unmarshal(data, &response); err != nil {
		log.WithError(err).WithField("data", string(data)).Error("Failed to parse SSE event data")
		return
	}

	// Log the parsed response
	log.WithFields(logrus.Fields{
		"event":  response.Event,
		"id":     response.ID,
		"answer": response.Answer,
	}).Info("Parsed SSE event")

	// Send to channel
	responseChan <- response
}
