package services

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

var emailConfig *EmailConfig

func InitEmail() {
	portStr := os.Getenv("SMTP_PORT")
	port := 587
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil {
			log.Printf("Warning: Invalid SMTP_PORT '%s', using default 587", portStr)
		} else {
			port = p
		}
	}
	emailConfig = &EmailConfig{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     port,
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     os.Getenv("SMTP_FROM"),
	}
	if emailConfig.Host == "" {
		log.Println("Warning: SMTP_HOST not configured. Email sending will be disabled.")
	}
}

func IsEmailConfigured() bool {
	return emailConfig != nil && emailConfig.Host != ""
}

func SendEmail(to, subject, body string, attachments []string) error {
	if !IsEmailConfigured() {
		return fmt.Errorf("SMTP not configured. Set SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD, SMTP_FROM in .env")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", emailConfig.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	for _, path := range attachments {
		m.Attach(path)
	}

	d := gomail.NewDialer(emailConfig.Host, emailConfig.Port, emailConfig.Username, emailConfig.Password)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
