/*
 * @file main.go
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @brief Check web certificates and create SendMail files
 * @version 0.1.0
 * @date 2026-11-02
 *
 * @copyright Copyright (c) 2026 ZHENG Robert
 *
 * SPDX-License-Identifier: MIT
 */

// CheckCert checks the TLS certificates of a list of websites. If a
// certificate expires within the warning threshold, a file
// <recipient>.SendMail is created in the output directory, which is then
// picked up and mailed by the SendMail application.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Zheng-Bote/check-cert/internal/config"
	"github.com/Zheng-Bote/check-cert/internal/controller"
	"github.com/Zheng-Bote/check-cert/internal/model"
	"github.com/Zheng-Bote/check-cert/internal/service"
)

// main is the entry point: loads the configuration, applies command line
// overrides, and runs the CheckController.
func main() {
	app := config.CheckCert
	fs := flag.NewFlagSet(app.ExecutableName, flag.ExitOnError)
	envFile := fs.String("c", "", "Configuration file (default: "+app.EnvFileName()+" next to the binary, otherwise in the current directory)")
	list := fs.String("l", "", "File with websites (overrides WEBSITES_FILE)")
	outDir := fs.String("o", "", "Output directory for the .SendMail file (overrides OUTPUT_DIR)")
	days := fs.Int("d", -1, "Warning threshold in days (overrides WARN_DAYS, default 5)")
	timeout := fs.Int("t", -1, "Timeout per connection in seconds (overrides TIMEOUT_SECONDS, default 10)")
	recipient := fs.String("r", "", "Recipient / file name (overrides MAIL_RECIPIENT)")
	showVersion := fs.Bool("v", false, "Show version")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "%s v%s - %s\n\nUsage: %s [options]\n\n",
			app.ProjectName, config.Version, app.ProgLongName, app.ExecutableName)
		fs.PrintDefaults()
		fmt.Fprintln(fs.Output(), "\nExit codes: 0 = ok, 1 = certificate expires soon, 2 = usage/configuration error, 3 = website could not be checked")
	}
	_ = fs.Parse(os.Args[1:])

	if *showVersion {
		fmt.Printf("%s v%s (build %s)\n%s\n", app.ProjectName, config.Version, config.BuildDate, config.ProjectHomepageURL)
		return
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unexpected arguments: %v\n", fs.Args())
		fs.Usage()
		os.Exit(controller.ExitUsage)
	}

	// 1. Load configuration (Model)
	cfg := model.NewCheckConfigModel()
	path := *envFile
	if path == "" {
		path = app.FindEnvFile()
	} else if _, err := os.Stat(path); err != nil {
		fmt.Fprintf(os.Stderr, "Error: configuration file %q not readable: %v\n", path, err)
		os.Exit(controller.ExitUsage)
	}
	if err := cfg.LoadEnv(path); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(controller.ExitUsage)
	}

	// 2. Command line overrides
	if *list != "" {
		cfg.WebsitesFile = *list
	}
	if *outDir != "" {
		cfg.OutputDir = *outDir
	}
	if *days >= 0 {
		cfg.WarnDays = *days
	}
	if *timeout > 0 {
		cfg.Timeout = time.Duration(*timeout) * time.Second
	}
	if *recipient != "" {
		cfg.MailRecipient = *recipient
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(controller.ExitUsage)
	}

	// 3. Service + Controller
	svc := service.NewCertService(cfg.Timeout)
	ctrl := controller.NewCheckController(cfg, svc, os.Stdout, os.Stderr)
	os.Exit(ctrl.Run())
}
