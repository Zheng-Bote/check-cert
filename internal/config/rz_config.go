/*
 * @file rz_config.go
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @brief Application information (Go counterpart of rz_config.hpp)
 * @version 0.2.0
 * @date 2026-11-02
 *
 * @copyright Copyright (c) 2026 ZHENG Robert
 *
 * SPDX-License-Identifier: MIT
 */

// Package config holds static application information.
// Version and BuildDate are injected at build time via -ldflags (see Makefile).
package config

import (
	"os"
	"path/filepath"
)

const (
	// ProjectHomepageURL is the project homepage.
	ProjectHomepageURL = "https://github.com/Zheng-Bote/check-cert"
	// Author is the program author.
	Author = "ZHENG Bote"
	// CreatedYear is the year of creation.
	CreatedYear = "2026"
	// Organization is the owning organization.
	Organization = "ZHENG Robert"
	// Domain is the organization domain.
	Domain = "net.hase-zheng"
)

var (
	// Version is the program version (overridden via -ldflags).
	Version = "0.2.0"
	// BuildDate is the build timestamp (overridden via -ldflags).
	BuildDate = "unknown"
)

// AppInfo describes one executable of this project.
type AppInfo struct {
	ProjectName        string // Short project name, also used for the .env file name.
	ProgLongName       string // Descriptive program name.
	ProjectDescription string // Purpose of the program.
	ExecutableName     string // Name of the binary.
}

// CheckCert is the application information of the certificate checker.
var CheckCert = AppInfo{
	ProjectName:        "CheckCert",
	ProgLongName:       "Check web certificate expiry",
	ProjectDescription: "Checks TLS certificates of websites and creates SendMail files",
	ExecutableName:     "CheckCert",
}

// SendMail is the application information of the mail dispatcher.
var SendMail = AppInfo{
	ProjectName:        "SendMail",
	ProgLongName:       "Send eMails based on input",
	ProjectDescription: "Watches a directory and mails the content of *.SendMail files",
	ExecutableName:     "SendMail",
}

// EnvFileName returns the name of the configuration file, e.g. "SendMail.env".
func (a AppInfo) EnvFileName() string {
	return a.ProjectName + ".env"
}

// FindEnvFile returns the path of the configuration file: next to the
// executable if present there, otherwise in the current working directory.
func (a AppInfo) FindEnvFile() string {
	name := a.EnvFileName()
	if exe, err := os.Executable(); err == nil {
		if real, err := filepath.EvalSymlinks(exe); err == nil {
			exe = real
		}
		p := filepath.Join(filepath.Dir(exe), name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return name
}
