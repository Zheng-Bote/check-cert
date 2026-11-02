/*
 * @file check_controller.go
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @brief Controller: reads the website list, runs the checks, writes the SendMail file
 * @version 0.1.0
 * @date 2026-11-02
 *
 * @copyright Copyright (c) 2026 ZHENG Robert
 *
 * SPDX-License-Identifier: MIT
 */

// Package controller orchestrates the certificate check workflow.
package controller

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Zheng-Bote/check-cert/internal/model"
	"github.com/Zheng-Bote/check-cert/internal/service"
)

// Exit codes of the program.
const (
	ExitOK          = 0 // All certificates are valid beyond the threshold.
	ExitWarning     = 1 // At least one certificate expires soon / has expired.
	ExitUsage       = 2 // Invalid invocation or configuration.
	ExitCheckFailed = 3 // At least one website could not be checked.
)

// CheckController runs the certificate checks.
type CheckController struct {
	cfg     *model.CheckConfigModel
	service *service.CertService
	out     io.Writer // "View": status output (stdout).
	errOut  io.Writer // "View": error output (stderr).
	now     func() time.Time
}

// NewCheckController creates a controller.
func NewCheckController(cfg *model.CheckConfigModel, svc *service.CertService, out, errOut io.Writer) *CheckController {
	return &CheckController{cfg: cfg, service: svc, out: out, errOut: errOut, now: time.Now}
}

// Run executes the full workflow and returns the exit code.
func (c *CheckController) Run() int {
	sites, err := c.readWebsites()
	if err != nil {
		fmt.Fprintf(c.errOut, "Error: %v\n", err)
		return ExitUsage
	}
	if len(sites) == 0 {
		fmt.Fprintf(c.errOut, "Error: website list %q contains no entries.\n", c.cfg.WebsitesFile)
		return ExitUsage
	}

	results := c.checkAll(sites)
	now := c.now()

	var warnings []string
	failed := false
	for _, r := range results {
		target := r.Website.Target()
		if r.Err != nil {
			fmt.Fprintf(c.errOut, "ERROR    %s: could not retrieve certificate (%v)\n", target, r.Err)
			failed = true
			continue
		}
		days := r.DaysLeft(now)
		end := r.NotAfter.UTC().Format("2006-01-02 15:04:05 MST")
		if days <= c.cfg.WarnDays {
			fmt.Fprintf(c.out, "WARNING  %s: %d days (%s)\n", target, days, end)
			warnings = append(warnings, r.MailText(now))
		} else {
			fmt.Fprintf(c.out, "OK       %s: %d days (%s)\n", target, days, end)
		}
	}

	if len(warnings) > 0 {
		path, err := c.writeMailFile(warnings)
		if err != nil {
			fmt.Fprintf(c.errOut, "Error: could not write SendMail file: %v\n", err)
			return ExitUsage
		}
		fmt.Fprintf(c.out, "SendMail file written: %s\n", path)
		return ExitWarning
	}
	if failed {
		return ExitCheckFailed
	}
	return ExitOK
}

// readWebsites parses the website list, skipping comments and blank lines.
func (c *CheckController) readWebsites() ([]model.Website, error) {
	f, err := os.Open(c.cfg.WebsitesFile)
	if err != nil {
		return nil, fmt.Errorf("website list not readable: %w", err)
	}
	defer f.Close()

	var sites []model.Website
	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		w, err := model.ParseWebsite(line)
		if err != nil {
			fmt.Fprintf(c.errOut, "Note: %s:%d skipped: %v\n", c.cfg.WebsitesFile, lineNo, err)
			continue
		}
		sites = append(sites, w)
	}
	return sites, sc.Err()
}

// checkAll checks all websites in parallel and returns the results in list order.
func (c *CheckController) checkAll(sites []model.Website) []model.CertResult {
	results := make([]model.CertResult, len(sites))
	jobs := make(chan int)
	var wg sync.WaitGroup

	workers := min(c.cfg.Workers, len(sites))
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				results[i] = c.service.Check(sites[i])
			}
		}()
	}
	for i := range sites {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return results
}

// writeMailFile writes all warnings into <OutputDir>/<recipient>.SendMail.
// An existing (not yet sent) file is extended. The file is first written
// to a temporary name and then renamed atomically, so a directory watcher
// (SendMail) never picks up a half-written file.
func (c *CheckController) writeMailFile(lines []string) (string, error) {
	if err := os.MkdirAll(c.cfg.OutputDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(c.cfg.OutputDir, c.cfg.MailFileName())

	var content strings.Builder
	if old, err := os.ReadFile(path); err == nil {
		content.Write(old)
		if len(old) > 0 && old[len(old)-1] != '\n' {
			content.WriteByte('\n')
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	for _, l := range lines {
		content.WriteString(l)
		content.WriteByte('\n')
	}

	tmp, err := os.CreateTemp(c.cfg.OutputDir, "."+c.cfg.MailFileName()+".*.tmp")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	if _, err := tmp.WriteString(content.String()); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return "", err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		os.Remove(tmpName)
		return "", err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return "", err
	}
	return path, nil
}
