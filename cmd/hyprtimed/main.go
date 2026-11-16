package main

import (
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/ayanrajpoot10/hyprtime/internal/db"
	"github.com/ayanrajpoot10/hyprtime/internal/lock"
	"github.com/ayanrajpoot10/hyprtime/internal/logger"
	"github.com/ayanrajpoot10/hyprtime/internal/tracker"
)

func main() {
	debug := flag.Bool("debug", false, "Enable debug logging")
	silent := flag.Bool("silent", false, "Silent all non-error logs")
	flag.Parse()

	logger.Init()

	if *debug {
		logger.SetLevel(logger.Debug)
	} else if *silent {
		logger.SetLevel(logger.Silent)
	}

	lk, err := lock.Acquire("hyprtimed")
	if err != nil {
		if errors.Is(err, lock.ErrAlreadyRunning) {
			logger.Infof("Another hyprtimed instance is already running, exiting")
			return
		}
		logger.Fatalf("Lock error: %v", err)
	}
	defer lk.Release()

	logger.Infof("Starting Daemon")

	db, err := db.New()
	if err != nil {
		logger.Fatalf("DB error: %v", err)
	}
	defer db.Close()

	t := tracker.New(db)
	if err := t.Start(); err != nil {
		logger.Fatalf("Tracker error: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	logger.Infof("Shutting down")

	t.Stop()

	logger.Infof("Daemon stopped")
}
