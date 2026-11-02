/*
 * @file cert_service.go
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @brief Service: retrieves the TLS certificate of a website
 * @version 0.1.0
 * @date 2026-11-02
 *
 * @copyright Copyright (c) 2026 ZHENG Robert
 *
 * SPDX-License-Identifier: MIT
 */

// Package service encapsulates the network access.
package service

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/Zheng-Bote/check-cert/internal/model"
)

// CertService fetches server certificates via TLS.
type CertService struct {
	timeout time.Duration
}

// NewCertService creates a CertService with the given connection timeout.
func NewCertService(timeout time.Duration) *CertService {
	return &CertService{timeout: timeout}
}

// Check connects to the website (with SNI) and returns the expiry date of
// the leaf certificate. Verification is skipped on purpose so that the
// expiry date of already expired or otherwise invalid certificates can
// still be read (same behaviour as "openssl s_client").
func (s *CertService) Check(w model.Website) model.CertResult {
	res := model.CertResult{Website: w}

	dialer := &net.Dialer{Timeout: s.timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", w.Target(), &tls.Config{
		ServerName:         w.Host,
		InsecureSkipVerify: true, //nolint:gosec // only the expiry date is evaluated
	})
	if err != nil {
		res.Err = err
		return res
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		res.Err = fmt.Errorf("no certificate received")
		return res
	}
	res.NotAfter = certs[0].NotAfter
	return res
}
