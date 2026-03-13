package v2

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine, apiVersion string, deps *Dependencies) {
	api := router.Group("/api/v2")
	api.Use(checkClientVersion(apiVersion))

	auth := api.Group("/auth")
	auth.POST("/signup", deps.AuthHandler.SignUp)
	auth.POST("/login", deps.AuthHandler.Login)
	auth.POST("/logout", deps.AuthHandler.Logout)

	protected := api.Group("")
	protected.Use(deps.AuthMiddleware.RequireAuth)

	users := protected.Group("/users")
	users.GET("/me", deps.UsersHandler.Me)

	recordings := protected.Group("/recordings")
	recordings.GET("", deps.RecordingsHandler.List)
	recordings.GET("/:id", deps.RecordingsHandler.Get)
	recordings.POST("/:id/preview-jobs", deps.RecordingsHandler.CreatePreviewJob)
	recordings.POST("/:id/analysis-jobs", deps.RecordingsHandler.CreateAnalysisJob)

	jobs := protected.Group("/jobs")
	jobs.GET("", deps.JobsHandler.List)
	jobs.GET("/:id", deps.JobsHandler.Get)

	analysis := protected.Group("/analysis")
	analysis.GET("/:id", deps.AnalysisHandler.Get)
}

func checkClientVersion(apiVersion string) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientVersion := c.GetHeader("X-API-Version")
		if clientVersion == "" {
			if version, exists := c.GetQuery("ApiVersion"); exists {
				clientVersion = version
			}
		}
		if clientVersion != apiVersion {
			c.AbortWithStatusJSON(http.StatusPreconditionFailed, gin.H{
				"error": fmt.Sprintf("client API version %s incompatible with server API version %s", clientVersion, apiVersion),
			})
			return
		}
		c.Next()
	}
}
