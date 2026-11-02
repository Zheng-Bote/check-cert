/*
 * @file smtp_service_test.go
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @brief Tests SmtpService against a fake SMTP server (STARTTLS + AUTH LOGIN)
 * @version 0.2.0
 * @date 2026-11-02
 *
 * @copyright Copyright (c) 2026 ZHENG Robert
 *
 * SPDX-License-Identifier: MIT
 */

package service

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"net"
	"net/mail"
	"net/textproto"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Zheng-Bote/check-cert/internal/model"
)

// selfSignedCert creates a TLS certificate for 127.0.0.1.
func selfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

type received struct {
	user, pass, from, to, data string
	tls                        bool
}

// fakeSMTP serves exactly one session and reports what it received.
func fakeSMTP(t *testing.T, ln net.Listener, cert tls.Certificate, out chan<- received) {
	conn, err := ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	var r received
	tp := textproto.NewConn(conn)
	reply := func(s string) { _ = tp.PrintfLine("%s", s) }
	reply("220 fake ESMTP")
	for {
		line, err := tp.ReadLine()
		if err != nil {
			out <- r
			return
		}
		cmd := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(cmd, "EHLO"):
			if r.tls {
				reply("250-fake\r\n250 AUTH PLAIN LOGIN")
			} else {
				reply("250-fake\r\n250 STARTTLS")
			}
		case cmd == "STARTTLS":
			reply("220 go ahead")
			tc := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{cert}})
			if err := tc.Handshake(); err != nil {
				t.Errorf("handshake: %v", err)
				return
			}
			conn = tc
			tp = textproto.NewConn(tc)
			r.tls = true
		case cmd == "AUTH LOGIN":
			reply("334 " + base64.StdEncoding.EncodeToString([]byte("Username:")))
			u, _ := tp.ReadLine()
			reply("334 " + base64.StdEncoding.EncodeToString([]byte("Password:")))
			p, _ := tp.ReadLine()
			ub, _ := base64.StdEncoding.DecodeString(u)
			pb, _ := base64.StdEncoding.DecodeString(p)
			r.user, r.pass = string(ub), string(pb)
			reply("235 ok")
		case strings.HasPrefix(cmd, "MAIL FROM:"):
			r.from = line[10:]
			reply("250 ok")
		case strings.HasPrefix(cmd, "RCPT TO:"):
			r.to = line[8:]
			reply("250 ok")
		case cmd == "DATA":
			reply("354 go")
			lines, _ := tp.ReadDotLines()
			r.data = strings.Join(lines, "\n")
			reply("250 queued")
		case cmd == "QUIT":
			reply("221 bye")
			out <- r
			return
		default:
			reply("500 unknown")
		}
	}
}

func TestSendStartTLSLogin(t *testing.T) {
	cert := selfSignedCert(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	out := make(chan received, 1)
	go fakeSMTP(t, ln, cert, out)

	port, _ := strconv.Atoi(strings.Split(ln.Addr().String(), ":")[1])
	cfg := model.NewSmtpConfigModel()
	cfg.Server, cfg.Port = "127.0.0.1", port
	cfg.Username, cfg.Password = "admin@example.com", "geheim"
	cfg.Timeout = 5 * time.Second

	pool := x509.NewCertPool()
	leaf, _ := x509.ParseCertificate(cert.Certificate[0])
	pool.AddCert(leaf)
	svc := NewSmtpService(cfg)
	svc.tlsConfig = &tls.Config{ServerName: "127.0.0.1", RootCAs: pool}

	msg := model.MailMessage{
		From:    &mail.Address{Address: "admin@example.com"},
		To:      &mail.Address{Address: "zheng.bote+webcertificate@googlemail.com"},
		Subject: "Certificate expires ✓",
		Body:    "The web certificate of the website hase-zheng.net expires in 4 days.\n.\nEnd",
	}
	if err := svc.Send(msg); err != nil {
		t.Fatalf("Send: %v", err)
	}

	r := <-out
	if !r.tls || r.user != "admin@example.com" || r.pass != "geheim" {
		t.Errorf("auth/tls wrong: %+v", r)
	}
	if r.from != "<admin@example.com>" || r.to != "<zheng.bote+webcertificate@googlemail.com>" {
		t.Errorf("envelope wrong: from=%q to=%q", r.from, r.to)
	}
	for _, want := range []string{
		"Subject: =?utf-8?q?Certificate_expires_=E2=9C=93?=",
		"Content-Type: text/plain; charset=UTF-8",
		"expires in 4 days.",
		"\n.\nEnd", // dot-stuffing must be transparent
	} {
		if !strings.Contains(r.data, want) {
			t.Errorf("data does not contain %q:\n%s", want, r.data)
		}
	}
}

func TestSendRefusesMissingStartTLS(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		w := bufio.NewWriter(c)
		r := bufio.NewReader(c)
		w.WriteString("220 fake\r\n")
		w.Flush()
		r.ReadString('\n')
		w.WriteString("250 fake\r\n") // no STARTTLS advertised
		w.Flush()
		r.ReadString('\n')
	}()
	port, _ := strconv.Atoi(strings.Split(ln.Addr().String(), ":")[1])
	cfg := model.NewSmtpConfigModel()
	cfg.Server, cfg.Port, cfg.Timeout = "127.0.0.1", port, 2*time.Second
	err := NewSmtpService(cfg).Send(model.MailMessage{
		From: &mail.Address{Address: "a@example.com"}, To: &mail.Address{Address: "b@example.com"},
	})
	if err == nil || !strings.Contains(err.Error(), "STARTTLS") {
		t.Fatalf("expected STARTTLS error, got %v", err)
	}
}
