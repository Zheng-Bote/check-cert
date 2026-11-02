/*
 * @file mail_controller.go
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @brief Controller: watches a directory and mails *.SendMail files
 * @version 0.2.0
 * @date 2026-11-02
 *
 * @copyright Copyright (c) 2026 ZHENG Robert
 *
 * SPDX-License-Identifier: MIT
 */

package controller

import (
	"context"
	"log"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Zheng-Bote/check-cert/internal/model"
)

// Sender delivers a mail message (implemented by service.SmtpService).
type Sender interface {
	Send(msg model.MailMessage) error
}

// failedFile remembers a failed attempt so the file is not retried on
// every scan (which would hammer the SMTP server, e.g. on wrong passwords).
type failedFile struct {
	modTime time.Time
	size    int64
	nextTry time.Time
}

// MailController watches WATCH_DIR for "<recipient>.SendMail" files,
// mails their content to <recipient> and deletes them after success.
type MailController struct {
	cfg    *model.SmtpConfigModel
	sender Sender
	from   *mail.Address
	log    *log.Logger // "View": status output.
	now    func() time.Time
	failed map[string]failedFile
}

// NewMailController creates a MailController. cfg must be validated.
func NewMailController(cfg *model.SmtpConfigModel, sender Sender, logger *log.Logger) *MailController {
	from, _ := mail.ParseAddress(cfg.From)
	return &MailController{
		cfg:    cfg,
		sender: sender,
		from:   from,
		log:    logger,
		now:    time.Now,
		failed: make(map[string]failedFile),
	}
}

// Run scans the watch directory immediately and then every POLL_SECONDS
// until ctx is cancelled. Polling is used deliberately: it works reliably
// on every file system (including network shares and removable media).
func (m *MailController) Run(ctx context.Context) {
	m.log.Printf("Watching directory: %s (every %s, minimum file age %s)", m.cfg.WatchDir, m.cfg.Poll, m.cfg.Stable)
	ticker := time.NewTicker(m.cfg.Poll)
	defer ticker.Stop()
	for {
		m.Scan()
		select {
		case <-ctx.Done():
			m.log.Println("Stopping monitoring.")
			return
		case <-ticker.C:
		}
	}
}

// Scan processes all eligible *.SendMail files once and returns the number
// of files that are still pending (too young, failed or invalid).
func (m *MailController) Scan() int {
	entries, err := os.ReadDir(m.cfg.WatchDir)
	if err != nil {
		m.log.Printf("ERROR: directory %s not readable: %v", m.cfg.WatchDir, err)
		return 0
	}

	pending := 0
	seen := make(map[string]bool)
	now := m.now()
	for _, e := range entries {
		name := e.Name()
		if !e.Type().IsRegular() || strings.HasPrefix(name, ".") || !strings.HasSuffix(name, model.MailFileSuffix) {
			continue
		}
		path := filepath.Join(m.cfg.WatchDir, name)
		seen[path] = true

		info, err := e.Info()
		if err != nil {
			continue // vanished in the meantime
		}
		// Stabilization: wait until the writer has finished.
		if now.Sub(info.ModTime()) < m.cfg.Stable {
			pending++
			continue
		}
		// Back-off after a failure, unless the file was changed since.
		if f, ok := m.failed[path]; ok && f.modTime.Equal(info.ModTime()) && f.size == info.Size() && now.Before(f.nextTry) {
			pending++
			continue
		}
		if !m.process(path, name, info) {
			pending++
		}
	}

	for path := range m.failed {
		if !seen[path] {
			delete(m.failed, path)
		}
	}
	return pending
}

// process sends one file and deletes it on success. Returns true on success.
func (m *MailController) process(path, name string, info os.FileInfo) bool {
	fail := func(format string, args ...any) bool {
		m.log.Printf("ERROR: "+format+" - retrying in %s", append(args, m.cfg.Retry)...)
		m.failed[path] = failedFile{modTime: info.ModTime(), size: info.Size(), nextTry: m.now().Add(m.cfg.Retry)}
		return false
	}

	to, err := model.RecipientFromFileName(name)
	if err != nil {
		return fail("%s: %v", name, err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return fail("%s not readable: %v", name, err)
	}

	msg := model.MailMessage{From: m.from, To: to, Subject: m.cfg.Subject, Body: string(body), Date: m.now()}
	m.log.Printf("Sending mail to %s (%d bytes) ...", to.Address, len(body))
	if err := m.sender.Send(msg); err != nil {
		return fail("mail to %s not sent: %v", to.Address, err)
	}
	m.log.Printf("Mail to %s sent successfully.", to.Address)
	delete(m.failed, path)

	// Only delete the file if it was not replaced/extended while sending;
	// otherwise keep it so the new content is not lost (it is sent again).
	cur, err := os.Stat(path)
	if err != nil {
		return true
	}
	if !os.SameFile(cur, info) || !cur.ModTime().Equal(info.ModTime()) || cur.Size() != info.Size() {
		m.log.Printf("Note: %s was modified while sending and is kept.", name)
		return true
	}
	if err := os.Remove(path); err != nil {
		m.log.Printf("ERROR: could not delete %s: %v", name, err)
		return true
	}
	m.log.Printf("Processed and deleted file %s.", name)
	return true
}
