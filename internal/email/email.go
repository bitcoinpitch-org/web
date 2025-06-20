package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"strconv"
	"time"
)

// Config holds the email configuration
type Config struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
	DevMode      bool // If true, log emails to console instead of sending
}

// Service handles email operations
type Service struct {
	config *Config
}

// NewService creates a new email service
func NewService(config *Config) *Service {
	if config.DevMode {
		log.Printf("Email Service: Running in DEV MODE - emails will be logged to console")
	} else {
		log.Printf("Email Service: Running in PRODUCTION MODE - emails will be sent via SMTP to %s", config.SMTPHost)
	}
	return &Service{
		config: config,
	}
}

// NewConfigFromEnv creates email config from environment variables
func NewConfigFromEnv() *Config {
	portStr := os.Getenv("SMTP_PORT")
	port := 587 // Default port
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	// Use dev mode only if SMTP credentials are missing, not just based on APP_ENV
	smtpHost := os.Getenv("SMTP_HOST")
	smtpUsername := os.Getenv("SMTP_USERNAME")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	smtpFromEmail := os.Getenv("SMTP_FROM_EMAIL")

	// Dev mode if any required SMTP setting is missing
	devMode := smtpHost == "" || smtpUsername == "" || smtpPassword == "" || smtpFromEmail == ""

	return &Config{
		SMTPHost:     smtpHost,
		SMTPPort:     port,
		SMTPUsername: smtpUsername,
		SMTPPassword: smtpPassword,
		FromEmail:    smtpFromEmail,
		FromName:     os.Getenv("SMTP_FROM_NAME"),
		DevMode:      devMode,
	}
}

// EmailData represents data for email templates
type EmailData struct {
	ToEmail         string
	ToName          string
	Subject         string
	SiteName        string
	SiteURL         string
	VerificationURL string
	ResetURL        string
	Username        string
	Token           string
}

// SendVerificationEmail sends an email verification email
func (s *Service) SendVerificationEmail(toEmail, toName, verificationURL string) error {
	data := EmailData{
		ToEmail:         toEmail,
		ToName:          toName,
		Subject:         "Verify your email - BitcoinPitch.org",
		SiteName:        "BitcoinPitch.org",
		SiteURL:         os.Getenv("SITE_URL"),
		VerificationURL: verificationURL,
	}

	htmlBody := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.Subject}}</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; padding: 20px; border-radius: 5px;">
        <h1 style="color: #f7931a;">{{.SiteName}}</h1>
        <h2>Verify Your Email Address</h2>
        <p>Hello{{if .ToName}} {{.ToName}}{{end}},</p>
        <p>Thank you for registering at BitcoinPitch.org! To complete your registration, please verify your email address by clicking the button below:</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.VerificationURL}}" style="background-color: #f7931a; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px; display: inline-block;">Verify Email</a>
        </div>
        <p>If the button doesn't work, you can copy and paste this link into your browser:</p>
        <p style="word-break: break-all;"><a href="{{.VerificationURL}}">{{.VerificationURL}}</a></p>
        <p><strong>This link will expire in 24 hours.</strong></p>
        <p>If you didn't create an account at BitcoinPitch.org, please ignore this email.</p>
        <hr style="margin: 30px 0; border: none; border-top: 1px solid #ddd;">
        <p style="color: #666; font-size: 12px;">
            This email was sent from BitcoinPitch.org<br>
            If you have any questions, please contact us.
        </p>
    </div>
</body>
</html>`

	return s.sendEmail(data, htmlBody)
}

// SendPasswordResetEmail sends a password reset email
func (s *Service) SendPasswordResetEmail(toEmail, toName, resetURL string) error {
	data := EmailData{
		ToEmail:  toEmail,
		ToName:   toName,
		Subject:  "Reset your password - BitcoinPitch.org",
		SiteName: "BitcoinPitch.org",
		SiteURL:  os.Getenv("SITE_URL"),
		ResetURL: resetURL,
	}

	htmlBody := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.Subject}}</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; padding: 20px; border-radius: 5px;">
        <h1 style="color: #f7931a;">{{.SiteName}}</h1>
        <h2>Password Reset Request</h2>
        <p>Hello{{if .ToName}} {{.ToName}}{{end}},</p>
        <p>We received a request to reset your password for your BitcoinPitch.org account. Click the button below to reset your password:</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.ResetURL}}" style="background-color: #f7931a; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px; display: inline-block;">Reset Password</a>
        </div>
        <p>If the button doesn't work, you can copy and paste this link into your browser:</p>
        <p style="word-break: break-all;"><a href="{{.ResetURL}}">{{.ResetURL}}</a></p>
        <p><strong>This link will expire in 1 hour.</strong></p>
        <p>If you didn't request a password reset, please ignore this email. Your password will not be changed.</p>
        <hr style="margin: 30px 0; border: none; border-top: 1px solid #ddd;">
        <p style="color: #666; font-size: 12px;">
            This email was sent from BitcoinPitch.org<br>
            If you have any questions, please contact us.
        </p>
    </div>
</body>
</html>`

	return s.sendEmail(data, htmlBody)
}

// sendEmail sends an email using SMTP or logs to console in dev mode
func (s *Service) sendEmail(data EmailData, htmlTemplate string) error {
	// In development mode, just log the email
	if s.config.DevMode {
		log.Printf("=== EMAIL (DEV MODE) ===")
		log.Printf("To: %s <%s>", data.ToName, data.ToEmail)
		log.Printf("Subject: %s", data.Subject)
		log.Printf("Verification URL: %s", data.VerificationURL)
		log.Printf("Reset URL: %s", data.ResetURL)
		log.Printf("=======================")
		return nil
	}

	// Parse and execute template
	tmpl, err := template.New("email").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	var htmlBody bytes.Buffer
	if err := tmpl.Execute(&htmlBody, data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	// Construct the email message
	from := s.config.FromEmail
	if s.config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromEmail)
	}

	// Format the date according to RFC 2822
	dateHeader := time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")

	msg := fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"Date: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"%s",
		data.ToEmail, from, data.Subject, dateHeader, htmlBody.String())

	// Set up authentication
	auth := smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)

	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         s.config.SMTPHost,
	}

	// Send the email
	address := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	// Connect to server
	c, err := smtp.Dial(address)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer c.Close()

	// Start TLS
	if err = c.StartTLS(tlsconfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Authenticate
	if err = c.Auth(auth); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	// Send email
	if err = c.Mail(s.config.FromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	if err = c.Rcpt(data.ToEmail); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err = w.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return c.Quit()
}
