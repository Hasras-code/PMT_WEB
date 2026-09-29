package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	stdmail "net/mail"
	"net/smtp"
	"strings"
	"time"
)

type SMTP struct {
	Addr, From         string
	Username, Password string
	TLSMode            string
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
	_, e = fmt.Fprintf(w, "From: %s\r\nTo: %s\r\nSubject: LMS %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nUse this one-time token for %s:\r\n%s\r\n", m.From, recipient.Address, purpose, purpose, token)
	if e != nil {
		return e
	}
	if e = w.Close(); e != nil {
		return e
	}
	return c.Quit()
}
