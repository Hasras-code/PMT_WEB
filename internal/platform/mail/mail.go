package mail

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type SMTP struct{ Addr, From string }

func (m SMTP) Send(ctx context.Context, to, purpose, token string) error {
	if strings.ContainsAny(to+m.From, "\r\n") {
		return fmt.Errorf("invalid email header")
	}
	conn, e := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp", m.Addr)
	if e != nil {
		return e
	}
	defer conn.Close()
	deadline := time.Now().Add(10 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if e = conn.SetDeadline(deadline); e != nil {
		return e
	}
	host, _, e := net.SplitHostPort(m.Addr)
	if e != nil {
		return e
	}
	c, e := smtp.NewClient(conn, host)
	if e != nil {
		return e
	}
	defer c.Close()
	if e = c.Mail(m.From); e != nil {
		return e
	}
	if e = c.Rcpt(to); e != nil {
		return e
	}
	w, e := c.Data()
	if e != nil {
		return e
	}
	_, e = fmt.Fprintf(w, "From: %s\r\nTo: %s\r\nSubject: LMS %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nUse this one-time token for %s:\r\n%s\r\n", m.From, to, purpose, purpose, token)
	if e != nil {
		return e
	}
	if e = w.Close(); e != nil {
		return e
	}
	return c.Quit()
}
