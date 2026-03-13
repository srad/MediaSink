package app

import (
	httpmiddleware "github.com/srad/mediasink/internal/http/middleware"
	v2 "github.com/srad/mediasink/internal/http/v2"
	analysissvc "github.com/srad/mediasink/internal/service/analysis"
	authsvc "github.com/srad/mediasink/internal/service/auth"
	jobsvc "github.com/srad/mediasink/internal/service/jobs"
	recordingssvc "github.com/srad/mediasink/internal/service/recordings"
	userssvc "github.com/srad/mediasink/internal/service/users"
	"github.com/srad/mediasink/internal/store/relational"
)

func initializeV2Dependencies(jwtSecret string) *v2.Dependencies {
	userStore := relational.NewUserStore()
	recordingStore := relational.NewRecordingStore()
	jobStore := relational.NewJobStore()
	analysisStore := relational.NewAnalysisStore()

	authService := authsvc.NewService(userStore, jwtSecret)
	usersService := userssvc.NewService(userStore)
	recordingsService := recordingssvc.NewService(recordingStore, jobStore)
	analysisService := analysissvc.NewService(recordingStore, jobStore, analysisStore)
	jobsService := jobsvc.NewService(jobStore)

	authMiddleware := httpmiddleware.NewAuthMiddleware(authService, usersService)

	return v2.NewDependencies(
		authMiddleware,
		v2.NewAuthHandler(authService),
		v2.NewUsersHandler(usersService),
		v2.NewRecordingsHandler(recordingsService, analysisService),
		v2.NewJobsHandler(jobsService),
		v2.NewAnalysisHandler(analysisService),
	)
}
