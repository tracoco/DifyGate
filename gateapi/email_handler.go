package gateapi

import (
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/tracoco/DifyGate/gate"
)

// EmailHandler handles email-related requests
type EmailHandler struct {
	mailService *gate.Service
	log         *logrus.Logger
}

// NewEmailHandler creates a new email handler
func NewEmailHandler(mailService *gate.Service, log *logrus.Logger) *EmailHandler {
	return &EmailHandler{
		mailService: mailService,
		log:         log,
	}
}

// SendEmailRequest represents the request body for sending an email
type SendEmailRequest struct {
	To          []string            `json:"to" binding:"required,min=1"`
	Cc          []string            `json:"cc,omitempty"`
	Bcc         []string            `json:"bcc,omitempty"`
	Subject     string              `json:"subject" binding:"required"`
	Body        string              `json:"body" binding:"required"`
	IsHTML      bool                `json:"is_html"`
	Attachments []AttachmentRequest `json:"attachments,omitempty"`
}

// AttachmentRequest represents email attachment data
type AttachmentRequest struct {
	Filename string `json:"filename" binding:"required"`
	Data     string `json:"data" binding:"required"` // base64 encoded
	MimeType string `json:"mime_type" binding:"required"`
}

// SendEmail handles the email sending endpoint
func (h *EmailHandler) SendEmail(c *gin.Context) {
	var req SendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert attachments if any
	attachments := []gate.Attachment{}
	for _, att := range req.Attachments {
		data, err := base64.StdEncoding.DecodeString(att.Data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid attachment data: " + err.Error(),
			})
			return
		}

		attachments = append(attachments, gate.Attachment{
			Filename: att.Filename,
			Data:     data,
			MimeType: att.MimeType,
		})
	}

	// Create email message
	msg := gate.Message{
		To:          req.To,
		Cc:          req.Cc,
		Bcc:         req.Bcc,
		Subject:     req.Subject,
		Body:        req.Body,
		IsHTML:      req.IsHTML,
		Attachments: attachments,
	}

	// Send the email
	if err := h.mailService.Send(msg); err != nil {
		h.log.WithError(err).Error("Failed to send email")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email sent successfully"})
}
