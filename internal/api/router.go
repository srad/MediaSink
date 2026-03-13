package api

import (
	"embed"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/srad/mediasink/config"
	"github.com/srad/mediasink/docs"
	"github.com/srad/mediasink/internal/app"
	"github.com/srad/mediasink/internal/middleware"
	"github.com/srad/mediasink/internal/ws"

	"github.com/gin-contrib/cors"
	v1 "github.com/srad/mediasink/internal/api/v1"

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
// @BasePath  /api/v2

// Setup InitRouter initialize routing information
func Setup(version, commit, apiVersion string, frontendFS embed.FS) http.Handler {
	router := gin.New()
	// r.Use(gin.Logger())
	router.Use(gin.Recovery())

	cfg := config.Read()

	// Add CORS headers specifically for static files to allow canvas usage
	router.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/videos/") {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Headers", "*")
			c.Header("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
		}
		c.Next()
	})

	router.Static("/videos", cfg.RecordingsAbsolutePath)

	// API V2
	docs.SwaggerInfo.BasePath = "/api/v2"
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

	apiV2 := router.Group("/api/v2")

	apiV2.Use(CheckClientVersion(apiVersion))

	apiV2.Use()
	{
		// Auth Group
		// ------------------------------------------------------
		auth := apiV2.Group("/auth")
		auth.POST("/signup", v1.CreateUser)
		auth.POST("/login", v1.Login)
		auth.POST("/logout", middleware.CheckAuthorizationHeader, v1.Logout)

		// User
		// ------------------------------------------------------
		user := apiV2.Group("/user")
		user.Use(middleware.CheckAuthorizationHeader)
		user.GET("/profile", v1.GetUserProfile)

		// Admin Group
		// ------------------------------------------------------
		admin := apiV2.Group("/admin")
		admin.Use(middleware.CheckAuthorizationHeader)
		admin.GET("/version", v1.GetVersion(version, commit))
		admin.POST("/import", v1.TriggerImport)
		admin.GET("/import", v1.GetImportInfo)

		// Channels Group
		// ------------------------------------------------------
		channels := apiV2.Group("/channels")
		channels.Use(middleware.CheckAuthorizationHeader)

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
		channels.POST("/:id/merge", v1.MergeVideos)
		// ------------------------------------------------------

		// Jobs Group
		// ------------------------------------------------------
		jobs := apiV2.Group("/jobs")
		jobs.Use(middleware.CheckAuthorizationHeader)

		jobs.POST("/:id", v1.AddPreviewJobs)
		jobs.POST("/stop/:pid", v1.StopJob)
		jobs.DELETE("/:id", v1.DestroyJob)
		jobs.POST("/list", v1.JobsList)
		jobs.POST("/resume", v1.ResumeJobs)
		jobs.POST("/pause", v1.PauseJobs)
		jobs.GET("/worker", v1.IsProcessing)

		// Recorder Group
		// ------------------------------------------------------
		recorder := apiV2.Group("/recorder")
		recorder.Use(middleware.CheckAuthorizationHeader)

		recorder.POST("/resume", v1.StartRecorder)
		recorder.POST("/pause", v1.StopRecorder)
		recorder.GET("", v1.IsRecording)

		// Videos Group
		// ------------------------------------------------------
		videos := apiV2.Group("/videos")
		videos.Use(middleware.CheckAuthorizationHeader)

		videos.POST("/updateinfo", v1.UpdateVideoInfo)
		videos.POST("/isupdating", v1.IsUpdatingVideoInfo)

		videos.GET("", v1.GetVideos)
		videos.POST("/filter", v1.FilterVideos)
		videos.GET("/random/:limit", v1.GetRandomVideos)
		videos.GET("/bookmarks", v1.GetBookmarkedVideos)
		videos.GET("/enhance/descriptions", v1.GetEnhancementDescriptions)
		videos.GET("/:id", v1.GetVideo)
		videos.GET("/:id/download", v1.DownloadVideo)

		videos.PATCH("/:id/fav", v1.FavVideo)
		videos.PATCH("/:id/unfav", v1.UnfavVideo)

		videos.POST("/:id/:mediaType/convert", v1.ConvertVideo)
		videos.POST("/:id/cut", v1.CutVideo)
		videos.POST("/:id/preview", v1.GenerateVideoPreviews)
		videos.POST("/:id/enhance", v1.EnhanceVideo)
		videos.POST("/:id/estimate-enhancement", v1.EstimateEnhancement)

		videos.DELETE("/:id", v1.DeleteVideo)

		// Previews Group
		// ------------------------------------------------------
		previews := apiV2.Group("/previews")
		previews.Use(middleware.CheckAuthorizationHeader)

		previews.POST("/regenerate", v1.RegenerateAllPreviews)
		previews.GET("/regenerate", v1.GetRegenerationProgress)

		// Analysis Group
		// ------------------------------------------------------
		analysis := apiV2.Group("/analysis")
		analysis.Use(middleware.CheckAuthorizationHeader)

		analysis.POST("/search/image", v1.SearchSimilarVideosByImage)
		analysis.POST("/group", v1.GroupSimilarVideos)
		analysis.POST("/all", v1.AnalyzeAllVideos)
		analysis.POST("/:id", v1.AnalyzeVideo)
		analysis.GET("/:id", v1.GetAnalysisResult)

		// Info Group
		// ------------------------------------------------------
		info := apiV2.Group("/info")
		info.Use(middleware.CheckAuthorizationHeader)

		info.GET("/:seconds", v1.GetInfo)
		info.GET("/disk", v1.GetDiskInfo)

		// Processes
		// ------------------------------------------------------
		apiV2.GET("/processes", middleware.CheckAuthorizationHeader, v1.GetProcesses)

		// WebSocket
		// ------------------------------------------------------
		go ws.WsListen()
		apiV2.GET("/ws", middleware.CheckAuthorizationHeader, ws.WsHandler)
	}

	serveFrontend(router, frontendFS, version, commit, apiVersion)

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
