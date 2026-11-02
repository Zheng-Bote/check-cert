/*
 * @file mail_controller_test.go
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @brief Tests the MailController with a fake sender
 * @version 0.2.0
 * @date 2026-11-02
 *
 * @copyright Copyright (c) 2026 ZHENG Robert
 *
 * SPDX-License-Identifier: MIT
 */

package controller

import (
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Zheng-Bote/check-cert/internal/model"
)

type fakeSender struct {
	sent []model.MailMessage
	err  error
}

func (f *fakeSender) Send(m model.MailMessage) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, m)
	return nil
}

func setup(t *testing.T) (*MailController, *fakeSender, string, *time.Time) {
	dir := t.TempDir()
	cfg := model.NewSmtpConfigModel()
	cfg.Server, cfg.From, cfg.WatchDir = "smtp.example.com", "admin@example.com", dir
	fs := &fakeSender{}
	c := NewMailController(cfg, fs, log.New(io.Discard, "", 0))
	now := time.Now()
	c.now = func() time.Time { return now }
	return c, fs, dir, &now
}

func write(t *testing.T, path, content string, age time.Duration, now time.Time) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	mt := now.Add(-age)
	if err := os.Chtimes(path, mt, mt); err != nil {
		t.Fatal(err)
	}
}

func TestScanSendsAndDeletes(t *testing.T) {
	c, fs, dir, now := setup(t)
	good := filepath.Join(dir, "zheng.bote+webcertificate@googlemail.com.SendMail")
	young := filepath.Join(dir, "young@example.com.SendMail")
	other := filepath.Join(dir, "ignore.txt")
	write(t, good, "Hallo\nWelt", 10*time.Second, *now)
	write(t, young, "zu jung", 0, *now)
	write(t, other, "x", time.Hour, *now)

	if pending := c.Scan(); pending != 1 {
		t.Errorf("pending = %d; want 1", pending)
	}
	if len(fs.sent) != 1 || fs.sent[0].To.Address != "zheng.bote+webcertificate@googlemail.com" ||
		fs.sent[0].Body != "Hallo\nWelt" || fs.sent[0].Subject != "New Message from SendMail" {
		t.Fatalf("unexpected sent: %+v", fs.sent)
	}
	if _, err := os.Stat(good); !os.IsNotExist(err) {
		t.Error("sent file was not deleted")
	}
	if _, err := os.Stat(young); err != nil {
		t.Error("young file must be kept")
	}
	if _, err := os.Stat(other); err != nil {
		t.Error("non-SendMail file must be kept")
	}
}

func TestScanRetryBackoff(t *testing.T) {
	c, fs, dir, now := setup(t)
	f := filepath.Join(dir, "a@example.com.SendMail")
	write(t, f, "x", 10*time.Second, *now)

	fs.err = errors.New("535 auth failed")
	c.Scan()
	fs.err = nil
	c.Scan() // within back-off: no new attempt
	if len(fs.sent) != 0 {
		t.Fatal("must not retry within back-off")
	}
	*now = now.Add(c.cfg.Retry + time.Second)
	c.Scan()
	if len(fs.sent) != 1 {
		t.Fatal("must retry after back-off")
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Error("file not deleted after successful retry")
	}
}

func TestScanInvalidRecipientKept(t *testing.T) {
	c, fs, dir, now := setup(t)
	f := filepath.Join(dir, "no-recipient.SendMail")
	write(t, f, "x", 10*time.Second, *now)
	if pending := c.Scan(); pending != 1 || len(fs.sent) != 0 {
		t.Fatalf("pending=%d sent=%d", pending, len(fs.sent))
	}
	if _, err := os.Stat(f); err != nil {
		t.Error("invalid file must be kept")
	}
}
