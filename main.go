package main

import (
	"context"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	serverapp "github.com/srad/mediasink/app"
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

	log.Infof("Version: %s, Commit: %s, Api Version %s", Version, Commit, ApiVersion)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	application, err := serverapp.InitializeApp(frontendFS, serverapp.Metadata{
		Version:    Version,
		Commit:     Commit,
		APIVersion: ApiVersion,
	})
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}

	if err := application.Run(ctx); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
