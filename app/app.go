package app

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/config"
	"github.com/srad/mediasink/internal/analysis/detectors/onnx"
	legacyapi "github.com/srad/mediasink/internal/api"
	"github.com/srad/mediasink/internal/db"
	"github.com/srad/mediasink/internal/services"
	"github.com/srad/mediasink/internal/store/vector"
)

type Metadata struct {
	Version    string
	Commit     string
	APIVersion string
}

type App struct {
	frontendFS embed.FS
	metadata   Metadata
	server     *http.Server
}

func InitializeApp(frontendFS embed.FS, metadata Metadata) (*App, error) {
	if err := validateEnvironment(); err != nil {
		return nil, err
	}

	db.Init()

	vectorStore := vector.NewSQLiteVecStore()
	vector.SetDefault(vectorStore)
	if err := vectorStore.Initialize(context.Background()); err != nil {
		return nil, fmt.Errorf("initialize vector store: %w", err)
	}

	if err := setupFolders(); err != nil {
		return nil, err
	}

	return &App{
		frontendFS: frontendFS,
		metadata:   metadata,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	services.StartUpJobs()
	services.StartRecorder()
	services.StartJobProcessing()

	gin.SetMode(gin.ReleaseMode)

	handler := legacyapi.Setup(a.metadata.Version, a.metadata.Commit, a.metadata.APIVersion, a.frontendFS)
	engine, ok := handler.(*gin.Engine)
	if !ok {
		return fmt.Errorf("unexpected router type %T", handler)
	}

	a.server = &http.Server{
		Addr:           "0.0.0.0:3000",
		Handler:        engine,
		ReadTimeout:    12 * time.Hour,
		WriteTimeout:   12 * time.Hour,
		MaxHeaderBytes: 0,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Infof("[app] start http server listening %s", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		return a.Shutdown(context.Background())
	case err := <-errCh:
		_ = a.Shutdown(context.Background())
		return err
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	var shutdownErr error

	if a.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		shutdownErr = a.server.Shutdown(shutdownCtx)
	}

	services.StopJobProcessing()
	services.StopRecorder()

	return shutdownErr
}

func validateEnvironment() error {
	if os.Getenv("SECRET") == "" {
		return fmt.Errorf("jwt SECRET environment variable is not set")
	}
	log.Infoln("OK: JWT SECRET environment variable is set.")

	cfg := config.Read()
	for _, path := range []string{cfg.DataDisk, cfg.RecordingsAbsolutePath} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("path %s does not exist", path)
		}
		log.Infof("Path %s exists.", path)
	}

	for _, name := range []string{"ffmpeg", "yt-dlp", "ffprobe"} {
		path, err := exec.LookPath(name)
		if err != nil {
			return fmt.Errorf("required executable %q not found in PATH: %w", name, err)
		}
		log.Debugf("OK: Found executable %q at %q", name, path)
	}

	if err := onnx.EnsureInitialized(); err != nil {
		return fmt.Errorf("failed to initialize ONNX runtime: %w", err)
	}
	if _, err := onnx.GetModelPath("mobilenet_v3_large"); err != nil {
		return fmt.Errorf("required ONNX model %q not found: %w", "mobilenet_v3_large", err)
	}
	log.Infoln("OK: ONNX runtime and models verified.")
	return nil
}

func setupFolders() error {
	channels, err := db.ChannelList()
	if err != nil {
		return err
	}
	for _, channel := range channels {
		if err := channel.ChannelName.MkDir(); err != nil {
			return err
		}
	}
	return nil
}
