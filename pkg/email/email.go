// Package email provides email senders for transactional auth messages.
// SMTPSender talks to a real SMTP server; LogSender just logs the link
// (development, when no SMTP is configured).
package email

import (
	"context"
	"fmt"
	"net/smtp"
	"net/url"
	"strconv"

	"go.uber.org/zap"
)

// SMTPSender sends email through an SMTP server (STARTTLS on submission ports).
type SMTPSender struct {
	addr        string // host:port
	host        string
	user        string
	pass        string
	fromAddress string
	fromName    string
	baseURL     string
}

// NewSMTPSender builds an SMTPSender.
func NewSMTPSender(host string, port int, user, pass, fromAddress, fromName, baseURL string) *SMTPSender {
	return &SMTPSender{
		addr:        host + ":" + strconv.Itoa(port),
		host:        host,
		user:        user,
		pass:        pass,
		fromAddress: fromAddress,
		fromName:    fromName,
		baseURL:     baseURL,
	}
}

// SendVerificationEmail sends an email-verification link.
func (s *SMTPSender) SendVerificationEmail(_ context.Context, to, token string) error {
	link := s.baseURL + "/verify-email?token=" + url.QueryEscape(token)
	body := "Verify your email address by opening this link:\r\n\r\n" + link +
		"\r\n\r\nIf you did not create an account, ignore this email."
	return s.send(to, "Verify your email", body)
}

// SendPasswordReset sends a password-reset link.
func (s *SMTPSender) SendPasswordReset(_ context.Context, to, token string) error {
	link := s.baseURL + "/reset-password?token=" + url.QueryEscape(token)
	body := "Reset your password by opening this link:\r\n\r\n" + link +
		"\r\n\r\nIf you did not request this, ignore this email."
	return s.send(to, "Reset your password", body)
}

func (s *SMTPSender) send(to, subject, body string) error {
	var auth smtp.Auth
	if s.user != "" {
		auth = smtp.PlainAuth("", s.user, s.pass, s.host)
	}
	msg := fmt.Sprintf(
		"From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\n"+
			"Content-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		s.fromName, s.fromAddress, to, subject, body,
	)
	if err := smtp.SendMail(s.addr, auth, s.fromAddress, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}

// LogSender logs the link instead of sending — for local development.
type LogSender struct {
	baseURL string
	log     *zap.Logger
}

// NewLogSender builds a LogSender.
func NewLogSender(baseURL string, log *zap.Logger) *LogSender {
	return &LogSender{baseURL: baseURL, log: log}
}

// SendVerificationEmail logs the verification link.
func (s *LogSender) SendVerificationEmail(_ context.Context, to, token string) error {
	s.log.Info("DEV email: verify email",
		zap.String("to", to),
		zap.String("link", s.baseURL+"/verify-email?token="+url.QueryEscape(token)),
	)
	return nil
}

// SendPasswordReset logs the reset link.
func (s *LogSender) SendPasswordReset(_ context.Context, to, token string) error {
	s.log.Info("DEV email: reset password",
		zap.String("to", to),
		zap.String("link", s.baseURL+"/reset-password?token="+url.QueryEscape(token)),
	)
	return nil
}
