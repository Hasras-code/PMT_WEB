package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html"
	"mime/multipart"
	"net"
	stdmail "net/mail"
	"net/smtp"
	"net/textproto"
	"net/url"
	"strings"
	"time"
)

type SMTP struct {
	Addr, From         string
	Username, Password string
	TLSMode            string
	FrontendURL        string
}

func (m SMTP) Send(ctx context.Context, to, purpose, token string) error {
	if strings.ContainsAny(to+m.From, "\r\n") {
		return fmt.Errorf("invalid email header")
	}
	from, e := stdmail.ParseAddress(m.From)
	if e != nil {
		return fmt.Errorf("invalid sender address")
	}
	recipient, e := stdmail.ParseAddress(to)
	if e != nil {
		return fmt.Errorf("invalid recipient address")
	}
	host, _, e := net.SplitHostPort(m.Addr)
	if e != nil {
		return fmt.Errorf("invalid SMTP address")
	}
	mode := strings.ToLower(m.TLSMode)
	if mode == "" {
		mode = "none"
	}
	if mode != "none" && mode != "starttls" && mode != "implicit" {
		return fmt.Errorf("invalid SMTP TLS mode")
	}
	if (m.Username != "" || m.Password != "") && (m.Username == "" || m.Password == "" || mode == "none") {
		return fmt.Errorf("SMTP authentication requires credentials and TLS")
	}

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, e := dialer.DialContext(ctx, "tcp", m.Addr)
	if e != nil {
		return e
	}
	deadline := time.Now().Add(10 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if e = conn.SetDeadline(deadline); e != nil {
		_ = conn.Close()
		return e
	}
	tlsConfig := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	if mode == "implicit" {
		tlsConn := tls.Client(conn, tlsConfig)
		if e = tlsConn.HandshakeContext(ctx); e != nil {
			_ = conn.Close()
			return fmt.Errorf("establish SMTP TLS: %w", e)
		}
		conn = tlsConn
	}
	c, e := smtp.NewClient(conn, host)
	if e != nil {
		_ = conn.Close()
		return e
	}
	defer c.Close()
	if mode == "starttls" {
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return fmt.Errorf("SMTP server does not support STARTTLS")
		}
		if e = c.StartTLS(tlsConfig); e != nil {
			return fmt.Errorf("establish SMTP STARTTLS: %w", e)
		}
	}
	if m.Username != "" || m.Password != "" {
		if ok, _ := c.Extension("AUTH"); !ok {
			return fmt.Errorf("SMTP server does not support authentication")
		}
		if e = c.Auth(smtp.PlainAuth("", m.Username, m.Password, host)); e != nil {
			return fmt.Errorf("authenticate SMTP: %w", e)
		}
	}
	if e = c.Mail(from.Address); e != nil {
		return e
	}
	if e = c.Rcpt(recipient.Address); e != nil {
		return e
	}
	w, e := c.Data()
	if e != nil {
		return e
	}
	message, e := m.message(from.Address, recipient.Address, purpose, token)
	if e != nil {
		_ = w.Close()
		return e
	}
	_, e = w.Write(message)
	if e != nil {
		return e
	}
	if e = w.Close(); e != nil {
		return e
	}
	return c.Quit()
}

func (m SMTP) message(from, to, purpose, token string) ([]byte, error) {
	subject := "PMT Family notification"
	textBody := fmt.Sprintf("Use this one-time token for %s:\r\n%s\r\n", purpose, token)
	htmlBody := fmt.Sprintf("<p>Use this one-time token for %s:</p><p><code>%s</code></p>", html.EscapeString(purpose), html.EscapeString(token))
	if purpose == "EMAIL_VERIFY" {
		frontend := strings.TrimRight(m.FrontendURL, "/")
		if frontend == "" {
			frontend = "http://localhost:5173"
		}
		link := frontend + "/verify#token=" + url.QueryEscape(token)
		subject = "Verify your PMT Family email"
		textBody = "Welcome to PMT Family.\r\n\r\nOpen this link to verify your email address:\r\n" + link + "\r\n\r\nIf the link does not open, paste this one-time token on the verification page:\r\n" + token + "\r\n"
		htmlBody = "<p>Welcome to PMT Family.</p><p><a href=\"" + html.EscapeString(link) + "\">Verify your email address</a></p><p>If the link does not open, paste this one-time token on the verification page:</p><p><code>" + html.EscapeString(token) + "</code></p>"
	}

	var body bytes.Buffer
	parts := multipart.NewWriter(&body)
	textPart, err := parts.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain; charset=utf-8"}, "Content-Transfer-Encoding": {"8bit"}})
	if err != nil {
		return nil, err
	}
	if _, err = textPart.Write([]byte(textBody)); err != nil {
		return nil, err
	}
	htmlPart, err := parts.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/html; charset=utf-8"}, "Content-Transfer-Encoding": {"8bit"}})
	if err != nil {
		return nil, err
	}
	if _, err = htmlPart.Write([]byte(htmlBody)); err != nil {
		return nil, err
	}
	if err = parts.Close(); err != nil {
		return nil, err
	}
	var message bytes.Buffer
	fmt.Fprintf(&message, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=%q\r\n\r\n", from, to, subject, parts.Boundary())
	_, err = message.Write(body.Bytes())
	return message.Bytes(), err
}
