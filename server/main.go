package main

import (
	"context"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	serverapp "github.com/srad/mediasink/server/app"
	"github.com/srad/mediasink/server/config"
)

var (
	Version    string
	Commit     string
	ApiVersion string
)

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: false,
	})

	// Read the environment exactly once, here, and pass the result down. The order
	// matters: the level has to be set before the warnings are emitted, or a
	// LOG_LEVEL of "fatal" would still print them.
	cfg, warnings, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}
	log.SetLevel(cfg.LogLevel)
	for _, warning := range warnings {
		log.Warn(warning)
	}

	log.Infof("Version: %s, Commit: %s, Api Version %s", Version, Commit, ApiVersion)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	application, err := serverapp.InitializeApp(frontendFS, serverapp.Metadata{
		Version:    Version,
		Commit:     Commit,
		APIVersion: ApiVersion,
	}, cfg)
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}

	if err := application.Run(ctx); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
