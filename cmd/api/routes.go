package main

import (
	"net/http"

	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /ping", app.ping)

	// Environments CRUD
	mux.HandleFunc("POST /v1/environments", app.createEnvironment)
	mux.HandleFunc("GET /v1/environments/{id}", app.getEnvironment)
	mux.HandleFunc("GET /v1/environments", app.getAllEnvironments)
	mux.HandleFunc("PUT /v1/environments/{id}", app.updateEnvironment)
	mux.HandleFunc("DELETE /v1/environments/{id}", app.deleteEnvironment)

	// Workers CR
	mux.HandleFunc("POST /v1/workers", app.createWorker)
	mux.HandleFunc("GET /v1/workers/{id}", app.getWorker)
	mux.HandleFunc("GET /v1/workers", app.getAllWorkers)

	standardChain := alice.New(app.recoverPanic, app.logRequests, app.enableCORS)

	return standardChain.Then(mux)
}
