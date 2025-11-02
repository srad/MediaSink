package handlers

import (
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// HandlerDependencies contains all dependencies required by job handlers
type HandlerDependencies struct {
	Logger log.FieldLogger
	DB     *gorm.DB
}

// NewHandlerDependencies creates a new HandlerDependencies instance
func NewHandlerDependencies(logger log.FieldLogger, db *gorm.DB) *HandlerDependencies {
	return &HandlerDependencies{
		Logger: logger,
		DB:     db,
	}
}
