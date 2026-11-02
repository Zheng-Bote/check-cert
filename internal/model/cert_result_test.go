/*
 * @file cert_result_test.go
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @brief Unit tests for the model package
 * @version 0.1.0
 * @date 2026-11-02
 *
 * @copyright Copyright (c) 2026 ZHENG Robert
 *
 * SPDX-License-Identifier: MIT
 */

package model

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseWebsite(t *testing.T) {
	cases := map[string]Website{
		"hase-zheng.net":                {"hase-zheng.net", "443"},
		"  bote.hase-zheng.net:443  ":   {"bote.hase-zheng.net", "443"},
		"example.com:8443":              {"example.com", "8443"},
		"https://www.example.org/a/b?x": {"www.example.org", "443"},
		"https://www.example.org:444/":  {"www.example.org", "444"},
		"[2001:db8::1]:443":             {"2001:db8::1", "443"},
		"[2001:db8::1]":                 {"2001:db8::1", "443"},
	}
	for in, want := range cases {
		got, err := ParseWebsite(in)
		if err != nil || got != want {
			t.Errorf("ParseWebsite(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := ParseWebsite("https:///"); err == nil {
		t.Error("expected error for empty host")
	}
}

func TestDaysLeftAndMailText(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	w := Website{"hase-zheng.net", "443"}

	r := CertResult{Website: w, NotAfter: now.Add(4*24*time.Hour + time.Hour)}
	if d := r.DaysLeft(now); d != 4 {
		t.Errorf("DaysLeft = %d; want 4", d)
	}
	want := `The web certificate of the website hase-zheng.net expires in 4 days. See also "openssl s_client -connect hase-zheng.net:443 </dev/null 2>/dev/null | openssl x509 -noout -enddate"`
	if got := r.MailText(now); got != want {
		t.Errorf("MailText =\n%s\nwant\n%s", got, want)
	}

	r.NotAfter = now.Add(-36 * time.Hour)
	if d := r.DaysLeft(now); d != -2 {
		t.Errorf("DaysLeft (expired) = %d; want -2", d)
	}
}

func TestLoadEnv(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, "CheckCert.env")
	content := "# comment\nWEBSITES_FILE=\"sites.txt\"\nOUTPUT_DIR=/abs/out\nWARN_DAYS='7'\nTIMEOUT_SECONDS=3 # inline\n"
	if err := os.WriteFile(env, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"WEBSITES_FILE", "OUTPUT_DIR", "WARN_DAYS", "TIMEOUT_SECONDS", "MAIL_RECIPIENT", "WORKERS"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}

	c := NewCheckConfigModel()
	if err := c.LoadEnv(env); err != nil {
		t.Fatal(err)
	}
	if c.WebsitesFile != filepath.Join(dir, "sites.txt") || c.OutputDir != "/abs/out" ||
		c.WarnDays != 7 || c.Timeout != 3*time.Second {
		t.Errorf("unexpected config: %+v", c)
	}
	if c.MailFileName() != "zheng.bote+webcertificate@googlemail.com.SendMail" {
		t.Errorf("MailFileName = %s", c.MailFileName())
	}
}
