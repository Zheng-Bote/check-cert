/*
 * @file smtp_config_model.go
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @brief Configuration Model of SendMail (SendMail.env)
 * @version 0.2.0
 * @date 2026-11-02
 *
 * @copyright Copyright (c) 2026 ZHENG Robert
 *
 * SPDX-License-Identifier: MIT
 */

package model

import (
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SmtpConfigModel holds the configuration of SendMail.
// The keys are compatible with the C++ SendMail application.
type SmtpConfigModel struct {
	Server      string        // SMTP_SERVER
	Port        int           // SMTP_PORT (default 587)
	Username    string        // SMTP_USERNAME (empty = no authentication)
	Password    string        // SMTP_PASSWORD
	From        string        // SMTP_FROM, e.g. "admin@example.com" or "Name <admin@example.com>"
	StartTLS    bool          // SMTP_STARTTLS (default true)
	ImplicitTLS bool          // SMTP_SSL (default true for port 465)
	Subject     string        // SMTP_SUBJECT
	Timeout     time.Duration // SMTP_TIMEOUT_SECONDS
	WatchDir    string        // WATCH_DIR
	Poll        time.Duration // POLL_SECONDS: scan interval of the watch directory
	Stable      time.Duration // STABLE_SECONDS: minimum file age before processing
	Retry       time.Duration // RETRY_SECONDS: wait time after a failed send
}

// NewSmtpConfigModel returns a SmtpConfigModel populated with default values.
func NewSmtpConfigModel() *SmtpConfigModel {
	return &SmtpConfigModel{
		Port:     587,
		StartTLS: true,
		Subject:  "New Message from SendMail",
		Timeout:  30 * time.Second,
		WatchDir: ".",
		Poll:     2 * time.Second,
		Stable:   2 * time.Second,
		Retry:    5 * time.Minute,
	}
}

// LoadEnv reads the .env file into the environment (existing environment
// variables take precedence) and applies the values to the model.
// A relative WATCH_DIR is resolved relative to the .env file's directory.
// A missing file is not an error.
func (c *SmtpConfigModel) LoadEnv(path string) error {
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

	get := func(key string) (string, bool) {
		v, ok := os.LookupEnv(key)
		return v, ok && v != ""
	}
	parseBool := func(key, v string) (bool, error) {
		switch strings.ToLower(v) {
		case "true", "1", "yes", "on":
			return true, nil
		case "false", "0", "no", "off":
			return false, nil
		}
		return false, fmt.Errorf("invalid %s: %q", key, v)
	}
	parseSeconds := func(key, v string, allowZero bool) (time.Duration, error) {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || (n == 0 && !allowZero) {
			return 0, fmt.Errorf("invalid %s: %q", key, v)
		}
		return time.Duration(n) * time.Second, nil
	}

	var err error
	if v, ok := get("SMTP_SERVER"); ok {
		c.Server = v
	}
	if v, ok := get("SMTP_PORT"); ok {
		if c.Port, err = strconv.Atoi(v); err != nil || c.Port <= 0 || c.Port > 65535 {
			return fmt.Errorf("invalid SMTP_PORT: %q", v)
		}
	}
	if v, ok := get("SMTP_USERNAME"); ok {
		c.Username = v
	}
	if v, ok := get("SMTP_PASSWORD"); ok {
		c.Password = v
	}
	if v, ok := get("SMTP_FROM"); ok {
		c.From = v
	}
	if v, ok := get("SMTP_STARTTLS"); ok {
		if c.StartTLS, err = parseBool("SMTP_STARTTLS", v); err != nil {
			return err
		}
	}
	c.ImplicitTLS = c.Port == 465
	if v, ok := get("SMTP_SSL"); ok {
		if c.ImplicitTLS, err = parseBool("SMTP_SSL", v); err != nil {
			return err
		}
	}
	if c.ImplicitTLS {
		c.StartTLS = false // already encrypted
	}
	if v, ok := get("SMTP_SUBJECT"); ok {
		c.Subject = v
	}
	if v, ok := get("SMTP_TIMEOUT_SECONDS"); ok {
		if c.Timeout, err = parseSeconds("SMTP_TIMEOUT_SECONDS", v, false); err != nil {
			return err
		}
	}
	if v, ok := get("WATCH_DIR"); ok {
		if !filepath.IsAbs(v) {
			v = filepath.Join(baseDir, v)
		}
		c.WatchDir = v
	}
	if v, ok := get("POLL_SECONDS"); ok {
		if c.Poll, err = parseSeconds("POLL_SECONDS", v, false); err != nil {
			return err
		}
	}
	if v, ok := get("STABLE_SECONDS"); ok {
		if c.Stable, err = parseSeconds("STABLE_SECONDS", v, true); err != nil {
			return err
		}
	}
	if v, ok := get("RETRY_SECONDS"); ok {
		if c.Retry, err = parseSeconds("RETRY_SECONDS", v, false); err != nil {
			return err
		}
	}
	return nil
}

// Validate checks that all mandatory values are present and consistent.
func (c *SmtpConfigModel) Validate() error {
	if c.Server == "" {
		return fmt.Errorf("SMTP_SERVER missing")
	}
	if c.From == "" {
		return fmt.Errorf("SMTP_FROM missing")
	}
	if _, err := mail.ParseAddress(c.From); err != nil {
		return fmt.Errorf("invalid SMTP_FROM: %q", c.From)
	}
	if c.Username != "" && c.Password == "" {
		return fmt.Errorf("SMTP_PASSWORD missing for SMTP_USERNAME %q", c.Username)
	}
	st, err := os.Stat(c.WatchDir)
	if err != nil || !st.IsDir() {
		return fmt.Errorf("WATCH_DIR %q is not a directory", c.WatchDir)
	}
	return nil
}

// Security returns a human readable description of the connection security.
func (c *SmtpConfigModel) Security() string {
	switch {
	case c.ImplicitTLS:
		return "SSL/TLS"
	case c.StartTLS:
		return "STARTTLS"
	default:
		return "UNENCRYPTED"
	}
}
