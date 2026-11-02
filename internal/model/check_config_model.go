/*
 * @file check_config_model.go
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @brief Configuration Model of CheckCert (CheckCert.env) and shared .env parser
 * @version 0.1.0
 * @date 2026-11-02
 *
 * @copyright Copyright (c) 2026 ZHENG Robert
 *
 * SPDX-License-Identifier: MIT
 */

// Package model contains the data structures of the application.
package model

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// CheckConfigModel holds the runtime configuration.
type CheckConfigModel struct {
	WebsitesFile  string        // Path to the list of websites.
	OutputDir     string        // Directory in which the .SendMail file is created.
	MailRecipient string        // Recipient; the file is named <MailRecipient>.SendMail.
	WarnDays      int           // Warning threshold in days.
	Timeout       time.Duration // Connection timeout per website.
	Workers       int           // Number of parallel checks.
}

// NewCheckConfigModel returns a CheckConfigModel populated with default values.
func NewCheckConfigModel() *CheckConfigModel {
	return &CheckConfigModel{
		WebsitesFile:  "websites.txt",
		OutputDir:     ".",
		MailRecipient: "zheng.bote+webcertificate@googlemail.com",
		WarnDays:      5,
		Timeout:       10 * time.Second,
		Workers:       8,
	}
}

// LoadEnv reads key/value pairs from an .env file into the process
// environment (existing environment variables take precedence) and then
// applies them to the model.
//
// Relative paths in the file are resolved relative to the file's directory,
// so the program works independently of the current working directory (cron).
// A missing file is not an error; defaults and environment variables are used.
func (c *CheckConfigModel) LoadEnv(path string) error {
	baseDir := "."
	if path != "" {
		if err := loadDotEnv(path); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		} else {
			baseDir = filepath.Dir(path)
		}
	}

	resolve := func(p string) string {
		if p == "" || filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(baseDir, p)
	}

	if v, ok := os.LookupEnv("WEBSITES_FILE"); ok && v != "" {
		c.WebsitesFile = resolve(v)
	}
	if v, ok := os.LookupEnv("OUTPUT_DIR"); ok && v != "" {
		c.OutputDir = resolve(v)
	}
	if v, ok := os.LookupEnv("MAIL_RECIPIENT"); ok && v != "" {
		c.MailRecipient = v
	}
	if v, ok := os.LookupEnv("WARN_DAYS"); ok && v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid WARN_DAYS: %q", v)
		}
		c.WarnDays = n
	}
	if v, ok := os.LookupEnv("TIMEOUT_SECONDS"); ok && v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return fmt.Errorf("invalid TIMEOUT_SECONDS: %q", v)
		}
		c.Timeout = time.Duration(n) * time.Second
	}
	if v, ok := os.LookupEnv("WORKERS"); ok && v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return fmt.Errorf("invalid WORKERS: %q", v)
		}
		c.Workers = n
	}
	return nil
}

// Validate checks the configuration for consistency.
func (c *CheckConfigModel) Validate() error {
	if c.WebsitesFile == "" {
		return fmt.Errorf("no website list specified")
	}
	if c.OutputDir == "" {
		return fmt.Errorf("no output directory specified")
	}
	if c.MailRecipient == "" || strings.ContainsAny(c.MailRecipient, "/\\") {
		return fmt.Errorf("invalid MAIL_RECIPIENT: %q", c.MailRecipient)
	}
	if c.WarnDays < 0 {
		return fmt.Errorf("warning threshold must not be negative")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}
	if c.Workers <= 0 {
		c.Workers = 1
	}
	return nil
}

// MailFileName returns the name of the SendMail file.
func (c *CheckConfigModel) MailFileName() string {
	return c.MailRecipient + ".SendMail"
}

// loadDotEnv parses a simple KEY="VALUE" file and sets variables that are
// not yet present in the environment. Lines starting with '#' are ignored.
func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s:%d: invalid line", path, lineNo)
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[len(val)-1] == val[0] {
			val = val[1 : len(val)-1]
		} else if i := strings.Index(val, " #"); i >= 0 {
			val = strings.TrimSpace(val[:i])
		}
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
	return sc.Err()
}
