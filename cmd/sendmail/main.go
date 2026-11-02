/*
 * @file main.go
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @brief Send mail based on input files in a watched directory
 * @version 0.2.0
 * @date 2026-11-02
 *
 * @copyright Copyright (c) 2026 ZHENG Robert
 *
 * SPDX-License-Identifier: MIT
 */

// SendMail watches a directory for files named "<recipient>.SendMail",
// mails their content to <recipient> and deletes them after success.
// It is the Go port of the C++/Qt SendMail application.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Zheng-Bote/check-cert/internal/config"
	"github.com/Zheng-Bote/check-cert/internal/controller"
	"github.com/Zheng-Bote/check-cert/internal/model"
	"github.com/Zheng-Bote/check-cert/internal/service"
)

// main is the entry point: loads the configuration, creates the
// SmtpService and MailController and starts monitoring.
func main() {
	app := config.SendMail
	fs := flag.NewFlagSet(app.ExecutableName, flag.ExitOnError)
	envFile := fs.String("c", "", "Configuration file (default: "+app.EnvFileName()+" next to the binary, otherwise in the current directory)")
	watchDir := fs.String("w", "", "Directory to watch (overrides WATCH_DIR)")
	once := fs.Bool("once", false, "Process existing files once and exit (e.g. for cron)")
	showVersion := fs.Bool("v", false, "Show version")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "%s v%s - %s\n\nUsage: %s [options] [directory]\n\n",
			app.ProjectName, config.Version, app.ProgLongName, app.ExecutableName)
		fs.PrintDefaults()
		fmt.Fprintln(fs.Output(), "\nExit codes: 0 = ok, 1 = (-once) not all files sent, 2 = usage/configuration error")
	}
	_ = fs.Parse(os.Args[1:])

	if *showVersion {
		fmt.Printf("%s v%s (build %s)\n%s\n", app.ProjectName, config.Version, config.BuildDate, config.ProjectHomepageURL)
		return
	}

	logger := log.New(os.Stderr, "", log.LstdFlags)
	logger.Printf("Starting %s v%s", app.ProgLongName, config.Version)

	// 1. Load configuration (Model)
	cfg := model.NewSmtpConfigModel()
	path := *envFile
	if path == "" {
		path = app.FindEnvFile()
	} else if _, err := os.Stat(path); err != nil {
		logger.Fatalf("Error: configuration file %q not readable: %v", path, err)
	}
	if err := cfg.LoadEnv(path); err != nil {
		logger.Printf("Error: %v", err)
		os.Exit(controller.ExitUsage)
	}

	// Override with args if provided (like the C++ version: first argument).
	switch {
	case *watchDir != "":
		cfg.WatchDir = *watchDir
	case fs.NArg() == 1:
		cfg.WatchDir = fs.Arg(0)
	case fs.NArg() > 1:
		logger.Printf("Error: too many arguments: %v", fs.Args())
		os.Exit(controller.ExitUsage)
	}
	if err := cfg.Validate(); err != nil {
		logger.Printf("Error: %v", err)
		os.Exit(controller.ExitUsage)
	}
	logger.Printf("SMTP server: %s:%d (%s), user: %s", cfg.Server, cfg.Port, cfg.Security(), cfg.Username)
	if cfg.Security() == "UNENCRYPTED" {
		logger.Println("WARNING: unencrypted connection!")
	}

	// 2. Service + Controller
	svc := service.NewSmtpService(cfg)
	ctrl := controller.NewMailController(cfg, svc, logger)

	if *once {
		if pending := ctrl.Scan(); pending > 0 {
			logger.Printf("%d file(s) not sent.", pending)
			os.Exit(controller.ExitWarning)
		}
		return
	}

	// 3. Start monitoring until SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctrl.Run(ctx)
}
