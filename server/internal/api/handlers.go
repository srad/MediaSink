package api

import (
	"github.com/srad/mediasink/server/config"
	v1 "github.com/srad/mediasink/server/internal/api/v1"
	"github.com/srad/mediasink/server/internal/db"
	"github.com/srad/mediasink/server/internal/middleware"
	"github.com/srad/mediasink/server/internal/services"
)

// Handlers is everything Setup needs to mount the route table. It grows one field per
// aggregate as each is converted; see ROADMAP.md.
type Handlers struct {
	Auth *v1.AuthHandler
	Gate *middleware.AuthMiddleware
}

// BuildHandlers wires stores into services into handlers.
//
// It lives here rather than in app/ because app imports this package, and the golden
// tests are in this package: putting it in app would make the test's import a cycle, and
// writing the graph out twice is how the goldens end up testing something the server does
// not do. The real composition root reclaims it in a later phase.
func BuildHandlers(handle *db.Handle, cfg config.Cfg) Handlers {
	userStore := db.NewUserStore(handle.Gorm())
	userService := services.NewUserService(userStore, cfg.JWTSecret)

	return Handlers{
		Auth: v1.NewAuthHandler(userService),
		Gate: middleware.NewAuthMiddleware(userService, cfg.JWTSecret),
	}
}
