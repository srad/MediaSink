package v2

import httpmiddleware "github.com/srad/mediasink/internal/http/middleware"

type Dependencies struct {
	AuthMiddleware    *httpmiddleware.AuthMiddleware
	AuthHandler       *AuthHandler
	UsersHandler      *UsersHandler
	RecordingsHandler *RecordingsHandler
	JobsHandler       *JobsHandler
	AnalysisHandler   *AnalysisHandler
}

func NewDependencies(
	authMiddleware *httpmiddleware.AuthMiddleware,
	authHandler *AuthHandler,
	usersHandler *UsersHandler,
	recordingsHandler *RecordingsHandler,
	jobsHandler *JobsHandler,
	analysisHandler *AnalysisHandler,
) *Dependencies {
	return &Dependencies{
		AuthMiddleware:    authMiddleware,
		AuthHandler:       authHandler,
		UsersHandler:      usersHandler,
		RecordingsHandler: recordingsHandler,
		JobsHandler:       jobsHandler,
		AnalysisHandler:   analysisHandler,
	}
}
