package handlers

import (
	"log"
	"muraadkov/finalgo/business/auth"
	"muraadkov/finalgo/business/mid"
	"muraadkov/finalgo/foundation/web"
	"net/http"
	"os"
)


func API(build string, shutdown chan os.Signal, log *log.Logger, a *auth.Auth) *web.App {

	app := web.NewApp(shutdown, mid.Logger(log), mid.Errors(log), mid.Metrics(), mid.Panics(log))

	check := check{
		log: log,
	}

	app.Handle(http.MethodGet, "/readiness", check.readiness, mid.Authenticate(a), mid.Authorize(auth.RoleAdmin))

	return app
}
