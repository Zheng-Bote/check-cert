/*
 * @file mail_message.go
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @brief Mail message model: recipient from file name, MIME construction
 * @version 0.2.0
 * @date 2026-11-02
 *
 * @copyright Copyright (c) 2026 ZHENG Robert
 *
 * SPDX-License-Identifier: MIT
 */

package model

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"time"
)

// MailFileSuffix is the suffix of files processed by SendMail.
const MailFileSuffix = ".SendMail"

// MailMessage is a plain text e-mail.
type MailMessage struct {
	From    *mail.Address
	To      *mail.Address
	Subject string
	Body    string
	Date    time.Time
}

// RecipientFromFileName extracts and validates the recipient address from a
// file name of the form "<address>.SendMail".
func RecipientFromFileName(name string) (*mail.Address, error) {
	if !strings.HasSuffix(name, MailFileSuffix) {
		return nil, fmt.Errorf("file name does not end with %s", MailFileSuffix)
	}
	raw := strings.TrimSuffix(name, MailFileSuffix)
	addr, err := mail.ParseAddress(raw)
	if err != nil || addr.Address != raw {
		return nil, fmt.Errorf("invalid recipient address %q", raw)
	}
	return addr, nil
}

// Bytes renders the message as RFC 5322 / MIME (UTF-8, quoted-printable).
func (m MailMessage) Bytes() ([]byte, error) {
	if m.From == nil || m.To == nil {
		return nil, fmt.Errorf("sender or recipient missing")
	}
	date := m.Date
	if date.IsZero() {
		date = time.Now()
	}

	var buf bytes.Buffer
	header := func(k, v string) { fmt.Fprintf(&buf, "%s: %s\r\n", k, v) }
	header("From", m.From.String())
	header("To", m.To.String())
	header("Subject", mime.QEncoding.Encode("utf-8", m.Subject))
	header("Date", date.Format(time.RFC1123Z))
	header("Message-ID", messageID(m.From.Address))
	header("MIME-Version", "1.0")
	header("Content-Type", "text/plain; charset=UTF-8")
	header("Content-Transfer-Encoding", "quoted-printable")
	buf.WriteString("\r\n")

	qp := quotedprintable.NewWriter(&buf)
	if _, err := qp.Write([]byte(strings.ReplaceAll(m.Body, "\r\n", "\n"))); err != nil {
		return nil, err
	}
	if err := qp.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// messageID creates a unique Message-ID using the domain of the sender.
func messageID(from string) string {
	domain := "localhost"
	if i := strings.LastIndex(from, "@"); i >= 0 && i < len(from)-1 {
		domain = from[i+1:]
	}
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), hex.EncodeToString(b), domain)
}
