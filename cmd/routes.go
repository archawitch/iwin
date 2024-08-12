package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w)
	})

	router.ServeFiles("/static/*filepath", http.Dir("ui/static"))

	router.HandlerFunc(http.MethodPost, "/addDevice", app.addDevice)
	router.HandlerFunc(http.MethodPost, "/connect", app.connect)
	router.HandlerFunc(http.MethodPost, "/upload", app.upload)

	local := alice.New(app.thisPCOnly)

	router.Handler(http.MethodGet, "/", local.ThenFunc(app.settings))
	router.Handler(http.MethodPost, "/settings", local.ThenFunc(app.settingsPost))
	router.Handler(http.MethodGet, "/devices", local.ThenFunc(app.devices))
	router.Handler(http.MethodPost, "/refresh", local.ThenFunc(app.refresh))
	router.Handler(http.MethodPost, "/verify", local.ThenFunc(app.verifyDevicePost))
	router.Handler(http.MethodPost, "/removeDevice", local.ThenFunc(app.removeDevice))

	middleware := alice.New(app.recoverPanic, app.logRequest, secureHeaders, app.clearPostFormData)
	
	return middleware.Then(router)
}