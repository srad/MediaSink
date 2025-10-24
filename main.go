package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/controllers"
	"github.com/srad/mediasink/database"
	"github.com/srad/mediasink/services"
)

var (
	Version    string
	Commit     string
	ApiVersion string
	cleanupOnce sync.Once
)

func init() {
	// 1. Env variables
	if os.Getenv("SECRET") == "" {
		log.Fatal("FATAL: JWT SECRET environment variable is not set.")
	}
	log.Infoln("OK: JWT SECRET environment variable is set.")

	// 2. File paths
	directories := []string{"/disk", "/recordings"}
	for _, path := range directories {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Fatalf("ERROR: Path %s does not exist.", path)
		} else {
			log.Infof("Path %s exists.", path)
		}
	}

	// 3. Check if needed executable exist
	executables := []string{"ffmpeg", "yt-dlp", "ffprobe"}
	for _, app := range executables {
		path, err := exec.LookPath(app)
		if err != nil {
			log.Fatalf("FATAL: Required executable '%s' not found in PATH: %v", app, err)
		}
		log.Infof("OK: Found executable '%s' at '%s'", app, path)
	}

	log.Infoln("All init checks passed.")
}

func main() {
	log.Infof("Version: %s, Commit: %s, Api Version %s", Version, Commit, ApiVersion)

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: false,
	})

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	database.Init()
	// models.StartMetrics(conf.AppCfg.NetworkDev)
	setupFolders()

	services.StartUpJobs()
	services.StartRecorder()
	services.StartJobProcessing()

	gin.SetMode("release")
	endPoint := fmt.Sprintf("0.0.0.0:%d", 3000)

	log.Infof("[main] start http server listening %s", endPoint)

	server := &http.Server{
		Addr:           endPoint,
		Handler:        controllers.Setup(Version, Commit, ApiVersion),
		ReadTimeout:    12 * time.Hour,
		WriteTimeout:   12 * time.Hour,
		MaxHeaderBytes: 0,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
		log.Infof("[main] start http server listening %s", endPoint)
	}()

	// Wait for signal and perform cleanup once
	<-c
	cleanupOnce.Do(func() {
		cleanup()
	})
	os.Exit(0)
}

func cleanup() {
	log.Infoln("cleanup ...")
	services.StopJobProcessing()
	services.StopRecorder()
	log.Infoln("cleanup complete")
}

func setupFolders() {
	channels, err := database.ChannelList()
	if err != nil {
		log.Errorln(err)
		return
	}
	for _, channel := range channels {
		if err := channel.ChannelName.MkDir(); err != nil {
			log.Errorln(err)
		}
	}
}
