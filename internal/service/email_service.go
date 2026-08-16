package service

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"

	"steam-download-tool/internal/config"
)

type EmailService struct {
	cfg *config.Config
}

func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{cfg: cfg}
}

// IsConfigured returns true if SMTP is properly configured.
func (s *EmailService) IsConfigured() bool {
	return s.cfg.SMTPHost != "" && s.cfg.SMTPUsername != "" && s.cfg.SMTPPassword != ""
}

// SendVerificationEmail sends a 6-digit verification code to the given email address.
func (s *EmailService) SendVerificationEmail(to, code string) error {
	if !s.IsConfigured() {
		return fmt.Errorf("SMTP is not configured")
	}

	from := s.cfg.SMTPFrom
	if from == "" {
		from = s.cfg.SMTPUsername
	}
	fromName := s.cfg.SMTPFromName
	if fromName == "" {
		fromName = "Steam Download Tool"
	}

	subject := "邮箱验证码 - Steam Download Tool"

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: Arial, sans-serif; padding: 20px;">
  <div style="max-width: 480px; margin: 0 auto; background: #f9f9f9; border-radius: 8px; padding: 32px;">
    <h2 style="color: #333; text-align: center;">邮箱验证码</h2>
    <p style="color: #666; text-align: center;">您正在注册 Steam Download Tool 账号，请输入以下验证码完成注册：</p>
    <div style="text-align: center; margin: 28px 0;">
      <span style="font-size: 36px; font-weight: bold; letter-spacing: 8px; color: #409eff; background: #ecf5ff; padding: 12px 24px; border-radius: 6px;">%s</span>
    </div>
    <p style="color: #999; font-size: 12px; text-align: center;">验证码 5 分钟内有效，请勿透露给他人。</p>
    <p style="color: #999; font-size: 12px; text-align: center;">如果这不是您的操作，请忽略此邮件。</p>
  </div>
</body>
</html>`, code)

	return s.sendMail(from, fromName, to, subject, body)
}

func (s *EmailService) sendMail(from, fromName, to, subject, body string) error {
	host := s.cfg.SMTPHost
	port := s.cfg.SMTPPort
	if port == "" {
		port = "587"
	}

	// Build headers
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", fromName, from)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	msg := ""
	for k, v := range headers {
		msg += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	msg += "\r\n" + body

	addr := net.JoinHostPort(host, port)
	auth := smtp.PlainAuth("", s.cfg.SMTPUsername, s.cfg.SMTPPassword, host)

	// Port 465: direct TLS connection (used by QQ mail, etc.)
	// Port 587: STARTTLS
	if port == "465" {
		tlsConfig := &tls.Config{
			ServerName: host,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("TLS connect to SMTP server: %w", err)
		}
		client, err := smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			return fmt.Errorf("create SMTP client: %w", err)
		}
		defer client.Close()

		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth: %w", err)
		}
		if err := client.Mail(from); err != nil {
			return fmt.Errorf("MAIL FROM: %w", err)
		}
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("RCPT TO: %w", err)
		}
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("DATA: %w", err)
		}
		_, err = fmt.Fprint(w, msg)
		if err != nil {
			return fmt.Errorf("write message: %w", err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("close writer: %w", err)
		}
		return client.Quit()
	}

	// Port 587 (default): STARTTLS
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("connect to SMTP server: %w", err)
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer client.Close()

	tlsConfig := &tls.Config{
		ServerName: host,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS: %w", err)
	}

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth: %w", err)
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}

	_, err = fmt.Fprint(w, msg)
	if err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("close writer: %w", err)
	}

	return client.Quit()
}

// sendMailNoTLS sends email without TLS (for testing or local servers, port 25).
func (s *EmailService) sendMailNoTLS(from, fromName, to, subject, body string) error {
	host := s.cfg.SMTPHost
	port := s.cfg.SMTPPort
	if port == "" {
		port = "25"
	}

	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", fromName, from)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	msg := ""
	for k, v := range headers {
		msg += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	msg += "\r\n" + body

	auth := smtp.PlainAuth("", s.cfg.SMTPUsername, s.cfg.SMTPPassword, host)
	addr := net.JoinHostPort(host, port)
	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}
