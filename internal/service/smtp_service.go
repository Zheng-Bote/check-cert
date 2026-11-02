/*
 * @file smtp_service.go
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @brief SMTP Service: delivers mails via STARTTLS / SSL with AUTH LOGIN or PLAIN
 * @version 0.2.0
 * @date 2026-11-02
 *
 * @copyright Copyright (c) 2026 ZHENG Robert
 *
 * SPDX-License-Identifier: MIT
 */

package service

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Zheng-Bote/check-cert/internal/model"
)

// SmtpService sends mails using the configured SMTP server.
type SmtpService struct {
	cfg       *model.SmtpConfigModel
	tlsConfig *tls.Config // optional override (tests)
}

// NewSmtpService creates a SmtpService.
func NewSmtpService(cfg *model.SmtpConfigModel) *SmtpService {
	return &SmtpService{cfg: cfg}
}

// Send delivers one message. Authentication prefers AUTH LOGIN (required
// by e.g. Netcup) and falls back to AUTH PLAIN. Credentials are never sent
// over an unencrypted connection (except to localhost).
func (s *SmtpService) Send(msg model.MailMessage) error {
	data, err := msg.Bytes()
	if err != nil {
		return err
	}

	addr := net.JoinHostPort(s.cfg.Server, strconv.Itoa(s.cfg.Port))
	tlsCfg := s.tlsConfig
	if tlsCfg == nil {
		tlsCfg = &tls.Config{ServerName: s.cfg.Server, MinVersion: tls.VersionTLS12}
	}

	dialer := &net.Dialer{Timeout: s.cfg.Timeout}
	var conn net.Conn
	if s.cfg.ImplicitTLS {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsCfg)
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("connection to %s failed: %w", addr, err)
	}
	_ = conn.SetDeadline(time.Now().Add(s.cfg.Timeout))

	c, err := smtp.NewClient(conn, s.cfg.Server)
	if err != nil {
		conn.Close()
		return fmt.Errorf("SMTP greeting failed: %w", err)
	}
	defer c.Close()

	if err := c.Hello(helloName()); err != nil {
		return fmt.Errorf("EHLO failed: %w", err)
	}

	if s.cfg.StartTLS {
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return errors.New("server does not support STARTTLS")
		}
		if err := c.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("STARTTLS failed: %w", err)
		}
	}

	if s.cfg.Username != "" {
		ok, mechs := c.Extension("AUTH")
		if !ok {
			return errors.New("server does not offer authentication")
		}
		list := strings.Fields(strings.ToUpper(mechs))
		var auth smtp.Auth
		switch {
		case slices.Contains(list, "LOGIN"):
			auth = &loginAuth{username: s.cfg.Username, password: s.cfg.Password, host: s.cfg.Server}
		case slices.Contains(list, "PLAIN"):
			auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Server)
		default:
			return fmt.Errorf("no supported AUTH mechanism (server offers: %s)", mechs)
		}
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	if err := c.Mail(msg.From.Address); err != nil {
		return fmt.Errorf("MAIL FROM rejected: %w", err)
	}
	if err := c.Rcpt(msg.To.Address); err != nil {
		return fmt.Errorf("RCPT TO rejected: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("DATA rejected: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		w.Close()
		return fmt.Errorf("sending the message failed: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("message rejected: %w", err)
	}
	_ = c.Quit()
	return nil
}

// helloName returns the host name used in EHLO.
func helloName() string {
	if h, err := os.Hostname(); err == nil && h != "" && !strings.ContainsAny(h, " \r\n") {
		return h
	}
	return "localhost"
}

// loginAuth implements the SMTP "AUTH LOGIN" mechanism.
type loginAuth struct {
	username, password, host string
}

// Start begins the AUTH LOGIN exchange; refuses unencrypted connections.
func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !server.TLS && !isLocalhost(server.Name) {
		return "", nil, errors.New("unencrypted connection, refusing to send credentials")
	}
	if server.Name != a.host {
		return "", nil, errors.New("wrong host name")
	}
	return "LOGIN", nil, nil
}

// Next answers the server's "Username:" / "Password:" challenges.
func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	switch strings.ToLower(strings.TrimSpace(string(fromServer))) {
	case "username:", "user name", "username":
		return []byte(a.username), nil
	case "password:", "password":
		return []byte(a.password), nil
	}
	return nil, fmt.Errorf("unexpected server challenge: %q", fromServer)
}

// isLocalhost reports whether name refers to the local machine.
func isLocalhost(name string) bool {
	return name == "localhost" || name == "127.0.0.1" || name == "::1"
}
