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
	responseChan := make(chan StreamingChatResponse)
	errChan := make(chan error, 1) // Buffered to avoid blocking

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
			h.log.WithField("dify_request", difyReq).Info("Sending Dify streaming request")
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

		// Send request
		client := &http.Client{}
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

		// Process the SSE stream
		reader := bufio.NewReader(resp.Body)
		for {
			// Check if context is done
			select {
			case <-ctx.Done():
				h.log.Info("Streaming context canceled")
				return
			default:
				// Continue processing
			}

			// Read a line from the stream
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					h.log.Info("Streaming response completed")
					return
				}
				h.log.WithError(err).Error("Error reading streaming response")
				errChan <- fmt.Errorf("error reading streaming response: %w", err)
				return
			}

			// Skip empty lines
			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				continue
			}

			// Check for data prefix
			if !bytes.HasPrefix(line, []byte("data: ")) {
				continue
			}

			// Extract the data part
			data := bytes.TrimPrefix(line, []byte("data: "))

			// Parse the data as JSON
			var streamResp StreamingChatResponse
			if err := json.Unmarshal(data, &streamResp); err != nil {
				h.log.WithError(err).Error("Failed to parse streaming response chunk")
				continue // Skip this chunk but continue processing
			}

			// Send the parsed response to the channel
			select {
			case responseChan <- streamResp:
				// Successfully sent
			case <-ctx.Done():
				h.log.Info("Streaming context canceled while sending response")
				return
			}
		}
	}()

	return responseChan, errChan
}
