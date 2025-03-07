package gate

import (
	"errors"
	"io"

	"github.com/sirupsen/logrus"
	gomail "gopkg.in/mail.v2"
)

// Attachment represents an email attachment
type Attachment struct {
	Filename string
	Data     []byte
	MimeType string
}

// Message represents an email message
type Message struct {
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	Body        string
	IsHTML      bool
	Attachments []Attachment
}

// DIFYGateConfig holds SMTP configuration
type DIFYGateConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	FromName string
}

// Service handles email operations
type Service struct {
	smtpHost     string
	smtpPort     int
	smtpUsername string
	smtpPassword string
	fromName     string
	log          *logrus.Logger
}

// NewService creates a new email service
func NewService(config DIFYGateConfig, log *logrus.Logger) *Service {
	return &Service{
		smtpHost:     config.Host,
		smtpPort:     config.Port,
		smtpUsername: config.Username,
		smtpPassword: config.Password,
		fromName:     config.FromName,
		log:          log,
	}
}

// Send sends an email
func (s *Service) Send(msg Message) error {
	if len(msg.To) == 0 {
		return errors.New("no recipients specified")
	}

	if s.smtpUsername == "" || s.smtpPassword == "" {
		return errors.New("SMTP credentials not configured")
	}

	m := gomail.NewMessage()

	// Set the sender with name if available
	from := s.smtpUsername
	if s.fromName != "" {
		from = m.FormatAddress(s.smtpUsername, s.fromName)
	}
	m.SetHeader("From", from)
	m.SetHeader("To", msg.To...)

	if len(msg.Cc) > 0 {
		m.SetHeader("Cc", msg.Cc...)
	}

	if len(msg.Bcc) > 0 {
		m.SetHeader("Bcc", msg.Bcc...)
	}

	m.SetHeader("Subject", msg.Subject)

	// Set body based on content type
	if msg.IsHTML {
		m.SetBody("text/html", msg.Body)
	} else {
		m.SetBody("text/plain", msg.Body)
	}

	// Add attachments
	for _, attachment := range msg.Attachments {
		m.Attach(attachment.Filename,
			gomail.SetCopyFunc(func(w io.Writer) error {
				_, err := w.Write(attachment.Data)
				return err
			}),
			gomail.SetHeader(map[string][]string{
				"Content-Type": {attachment.MimeType},
			}),
		)
	}

	// Configure the dialer with SMTP server settings
	d := gomail.NewDialer(s.smtpHost, s.smtpPort, s.smtpUsername, s.smtpPassword)

	// Send the email
	if err := d.DialAndSend(m); err != nil {
		s.log.WithError(err).Error("Failed to send email")
		return err
	}

	return nil
}
