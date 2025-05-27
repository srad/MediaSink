package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/srad/mediasink/app"
	"github.com/srad/mediasink/conf"
	"github.com/srad/mediasink/docs"
	"github.com/srad/mediasink/middlewares"
	"github.com/srad/mediasink/network"

	"github.com/gin-contrib/cors"
	v1 "github.com/srad/mediasink/controllers/api/v1"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           MediaSink API
// @version         1.0
// @description     The rest API of the server.
//
// @contact.name   API Support
// @contact.url    https://github.com/srad
//
// @license.name  Dual license, non-commercial, but free for open-source and educational uses.
//
// @BasePath  /api/v1

// Setup InitRouter initialize routing information
func Setup(version, commit, apiVersion string) http.Handler {
	router := gin.New()
	// r.Use(gin.Logger())
	router.Use(gin.Recovery())

	cfg := conf.Read()

	// This is only for development. User nginx or something to serve the static files.
	router.Static("/videos", cfg.RecordingsAbsolutePath)

	// API V1
	docs.SwaggerInfo.BasePath = "/api/v1"
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	router.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		AllowHeaders:     []string{"*", "Authorization", "Content-Type"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           7 * 24 * time.Hour,
		AllowWebSockets:  true,
		AllowWildcard:    true,
	}))

	apiV1 := router.Group("/api/v1")

	apiV1.Use(CheckClientVersion(apiVersion))

	apiV1.Use()
	{
		// Auth Group
		// ------------------------------------------------------
		auth := apiV1.Group("/auth")
		auth.POST("/signup", v1.CreateUser)
		auth.POST("/login", v1.Login)
		auth.POST("/logout", middlewares.CheckAuthorizationHeader, v1.Logout)

		// User
		// ------------------------------------------------------
		user := apiV1.Group("/user")
		user.Use(middlewares.CheckAuthorizationHeader)
		user.GET("/profile", v1.GetUserProfile)

		// Admin Group
		// ------------------------------------------------------
		admin := apiV1.Group("/admin")
		admin.Use(middlewares.CheckAuthorizationHeader)
		admin.GET("/version", v1.GetVersion(version, commit))
		admin.POST("/import", v1.TriggerImport)
		admin.GET("/import", v1.GetImportInfo)

		// Channels Group
		// ------------------------------------------------------
		channels := apiV1.Group("/channels")
		channels.Use(middlewares.CheckAuthorizationHeader)

		channels.GET("", v1.GetChannels)
		channels.POST("", v1.CreateChannel)

		channels.GET("/:id", v1.GetChannel)
		channels.DELETE("/:id", v1.DeleteChannel)
		channels.PATCH("/:id", v1.UpdateChannel)

		channels.POST("/:id/resume", v1.ResumeChannel)
		channels.POST("/:id/pause", v1.PauseChannel)

		channels.PATCH("/:id/fav", v1.FavChannel)
		channels.PATCH("/:id/unfav", v1.UnFavChannel)

		channels.POST("/:id/upload", v1.UploadChannel)
		channels.PATCH("/:id/tags", v1.TagChannel)
		// ------------------------------------------------------

		// Jobs Group
		// ------------------------------------------------------
		jobs := apiV1.Group("/jobs")
		jobs.Use(middlewares.CheckAuthorizationHeader)

		jobs.POST("/:id", v1.AddPreviewJobs)
		jobs.POST("/stop/:pid", v1.StopJob)
		jobs.DELETE("/:id", v1.DestroyJob)
		jobs.POST("/list", v1.JobsList)
		jobs.POST("/resume", v1.ResumeJobs)
		jobs.POST("/pause", v1.PauseJobs)
		jobs.GET("/worker", v1.IsProcessing)

		// Recorder Group
		// ------------------------------------------------------
		recorder := apiV1.Group("/recorder")
		recorder.Use(middlewares.CheckAuthorizationHeader)

		recorder.POST("/resume", v1.StartRecorder)
		recorder.POST("/pause", v1.StopRecorder)
		recorder.GET("", v1.IsRecording)

		// Videos Group
		// ------------------------------------------------------
		videos := apiV1.Group("/videos")
		videos.Use(middlewares.CheckAuthorizationHeader)

		videos.POST("/updateinfo", v1.UpdateVideoInfo)
		videos.POST("/isupdating", v1.IsUpdatingVideoInfo)
		videos.POST("/generate/posters", v1.GenerateCovers)

		videos.GET("", v1.GetVideos)
		videos.POST("/filter", v1.FilterVideos)
		videos.GET("/random/:limit", v1.GetRandomVideos)
		videos.GET("/bookmarks", v1.GetBookmarkedVideos)
		videos.GET("/:id", v1.GetVideo)
		videos.GET("/:id/download", v1.DownloadVideo)

		videos.PATCH("/:id/fav", v1.FavVideo)
		videos.PATCH("/:id/unfav", v1.UnfavVideo)

		videos.POST("/:id/:mediaType/convert", v1.ConvertVideo)
		videos.POST("/:id/cut", v1.CutVideo)
		videos.POST("/:id/preview", v1.GenerateVideoPreviews)

		videos.DELETE("/:id", v1.DeleteVideo)

		// Info Group
		// ------------------------------------------------------
		info := apiV1.Group("/info")
		info.Use(middlewares.CheckAuthorizationHeader)

		info.GET("/:seconds", v1.GetInfo)
		info.GET("/disk", v1.GetDiskInfo)

		// Processes
		// ------------------------------------------------------
		apiV1.GET("/processes", middlewares.CheckAuthorizationHeader, v1.GetProcesses)

		// WebSocket
		// ------------------------------------------------------
		go network.WsListen()
		apiV1.GET("/ws", middlewares.CheckAuthorizationHeader, network.WsHandler)
	}

	return router
}

func CheckClientVersion(apiVersion string) gin.HandlerFunc {
	return func(c *gin.Context) {
		appG := app.Gin{C: c}

		var clientVersion = c.GetHeader("X-API-Version")
		if clientVersion == "" {
			// WebSocket via get param?
			if version, exists := c.GetQuery("ApiVersion"); exists {
				clientVersion = version
			}
		}
		if clientVersion != apiVersion {
			appG.Error(http.StatusPreconditionFailed, fmt.Errorf("client API version %s incompatible with server API version %s", clientVersion, apiVersion))
			return
		}
		c.Next()
	}
}
