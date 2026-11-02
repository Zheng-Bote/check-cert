/*
 * @file cert_result.go
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @brief Result model of a single certificate check
 * @version 0.1.0
 * @date 2026-11-02
 *
 * @copyright Copyright (c) 2026 ZHENG Robert
 *
 * SPDX-License-Identifier: MIT
 */

package model

import (
	"fmt"
	"math"
	"net"
	"strings"
	"time"
)

// Website describes a host/port pair to check.
type Website struct {
	Host string
	Port string
}

// Target returns "host:port" (IPv6-safe).
func (w Website) Target() string {
	return net.JoinHostPort(w.Host, w.Port)
}

// ParseWebsite parses one line of the website list.
// Accepted formats: "host", "host:port", "https://host[:port]/path".
func ParseWebsite(line string) (Website, error) {
	s := strings.TrimSpace(line)
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	if s == "" {
		return Website{}, fmt.Errorf("empty entry")
	}

	host, port := s, "443"
	if h, p, err := net.SplitHostPort(s); err == nil {
		host, port = h, p
	} else if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		host = s[1 : len(s)-1]
	}
	if host == "" || port == "" {
		return Website{}, fmt.Errorf("invalid entry %q", line)
	}
	return Website{Host: host, Port: port}, nil
}

// CertResult is the outcome of checking one website.
type CertResult struct {
	Website  Website
	NotAfter time.Time
	Err      error
}

// DaysLeft returns the number of whole days until expiry
// (negative if the certificate has already expired).
func (r CertResult) DaysLeft(now time.Time) int {
	d := r.NotAfter.Sub(now).Hours() / 24
	if d >= 0 {
		return int(math.Floor(d))
	}
	return -int(math.Ceil(-d))
}

// MailText returns the notification text for this result.
func (r CertResult) MailText(now time.Time) string {
	days := r.DaysLeft(now)
	var text string
	if days < 0 {
		text = fmt.Sprintf("The web certificate of the website %s expired %d days ago.", r.Website.Host, -days)
	} else {
		text = fmt.Sprintf("The web certificate of the website %s expires in %d days.", r.Website.Host, days)
	}
	return fmt.Sprintf("%s See also \"openssl s_client -connect %s </dev/null 2>/dev/null | openssl x509 -noout -enddate\"",
		text, r.Website.Target())
}
