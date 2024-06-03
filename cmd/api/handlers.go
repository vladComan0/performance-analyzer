package main

import (
	"errors"
	"fmt"
	"github.com/vladComan0/performance-analyzer/internal/custom_errors"
	"github.com/vladComan0/performance-analyzer/internal/data"
	"github.com/vladComan0/performance-analyzer/internal/dto"
	"github.com/vladComan0/performance-analyzer/pkg/helpers"
	"net/http"
)

func (app *application) ping(w http.ResponseWriter, _ *http.Request) {
	if err := app.environmentService.PingDB(); err != nil {
		app.helper.ServerError(w, err)
		return
	}
	_, err := w.Write([]byte("pong"))
	if err != nil {
		app.helper.ServerError(w, err)
		return
	}
}

func (app *application) createEnvironment(w http.ResponseWriter, r *http.Request) {
	var input dto.CreateEnvironmentInput

	if err := app.helper.ReadJSON(w, r, &input); err != nil {
		app.helper.ClientError(w, http.StatusBadRequest)
		return
	}

	environment, err := app.environmentService.CreateEnvironment(input)
	if err != nil {
		app.helper.ServerError(w, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("v1/environments/%d", environment.ID))

	if err = app.helper.WriteJSON(w, http.StatusCreated, helpers.Envelope{"environment": environment}, headers); err != nil {
		app.helper.ServerError(w, err)
		return
	}

	app.log.Info().Msgf("Created new environment with id: %d", environment.ID)
}

func (app *application) getEnvironment(w http.ResponseWriter, r *http.Request) {
	id, err := app.helper.GetID(r)
	if err != nil || id < 1 {
		app.helper.ClientError(w, http.StatusBadRequest)
		return
	}

	environment, err := app.environmentService.GetEnvironment(id)
	if err != nil {
		switch {
		case errors.Is(err, custom_errors.ErrNoRecord):
			app.helper.ClientError(w, http.StatusNotFound)
		default:
			app.helper.ServerError(w, err)
		}
		return
	}

	if err = app.helper.WriteJSON(w, http.StatusOK, helpers.Envelope{"environment": environment}, nil); err != nil {
		app.helper.ServerError(w, err)
		return
	}
}

func (app *application) getAllEnvironments(w http.ResponseWriter, _ *http.Request) {
	environments, err := app.environmentService.GetEnvironments()
	if err != nil {
		switch {
		case errors.Is(err, custom_errors.ErrNoRecord):
			app.helper.ClientError(w, http.StatusNotFound)
		default:
			app.helper.ServerError(w, err)
		}
		return
	}

	app.log.Info().Msgf("Environments: %v", environments)

	if err = app.helper.WriteJSON(w, http.StatusOK, helpers.Envelope{"environments": environments}, nil); err != nil {
		app.helper.ServerError(w, err)
		return
	}

	app.log.Info().Msgf("Retrieved all environments")
}

func (app *application) updateEnvironment(w http.ResponseWriter, r *http.Request) {
	id, err := app.helper.GetID(r)
	if err != nil || id < 1 {
		app.helper.ClientError(w, http.StatusBadRequest)
		return
	}

	var input dto.UpdateEnvironmentInput
	if err := app.helper.ReadJSON(w, r, &input); err != nil {
		app.helper.ClientError(w, http.StatusBadRequest)
		return
	}

	updatedEnvironment, err := app.environmentService.UpdateEnvironment(id, input)
	if err != nil {
		switch {
		case errors.Is(err, custom_errors.ErrNoRecord):
			app.helper.ClientError(w, http.StatusNotFound)
		default:
			app.helper.ServerError(w, err)
		}
		return
	}

	if err := app.helper.WriteJSON(w, http.StatusOK, helpers.Envelope{"environment": updatedEnvironment}, nil); err != nil {
		app.helper.ServerError(w, err)
		return
	}
}

func (app *application) deleteEnvironment(w http.ResponseWriter, r *http.Request) {
	id, err := app.helper.GetID(r)
	if err != nil || id < 1 {
		app.helper.ClientError(w, http.StatusBadRequest)
		return
	}

	if err = app.environmentService.DeleteEnvironment(id); err != nil {
		switch {
		case errors.Is(err, custom_errors.ErrNoRecord):
			app.helper.ClientError(w, http.StatusNotFound)
		default:
			app.helper.ServerError(w, err)
		}
		return
	}

	if err := app.helper.WriteJSON(w, http.StatusOK, helpers.Envelope{"message": "Environment successfully deleted"}, nil); err != nil {
		app.helper.ServerError(w, err)
		return
	}

	app.log.Info().Msgf("Deleted environment with id: %d", id)
}

func (app *application) createWorker(w http.ResponseWriter, r *http.Request) {
	var input *data.Worker

	if err := app.helper.ReadJSON(w, r, &input); err != nil {
		app.helper.ClientError(w, http.StatusBadRequest)
		return
	}

	worker, err := app.workerService.CreateWorker(input)
	if err != nil {
		switch {
		case errors.Is(err, custom_errors.ErrInvalidInput):
			app.helper.ClientError(w, http.StatusBadRequest)
		case errors.Is(err, custom_errors.ErrEnvironmentDisabled):
			app.helper.ClientError(w, http.StatusForbidden)
		default:
			app.helper.ServerError(w, err)
		}
	}

	// Make the application aware of that new location -> add the headers to the right json helper function
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("v1/workers/%d", worker.ID))

	if err := app.helper.WriteJSON(w, http.StatusCreated, helpers.Envelope{"worker": worker}, nil); err != nil {
		app.helper.ServerError(w, err)
		return
	}

	app.log.Info().Msgf("Created new worker with id: %d", worker.ID)
}

func (app *application) getWorker(w http.ResponseWriter, r *http.Request) {
	id, err := app.helper.GetID(r)
	if err != nil {
		app.helper.ClientError(w, http.StatusBadRequest)
		return
	}

	worker, err := app.workerService.GetWorker(id)
	if err != nil {
		switch {
		case errors.Is(err, custom_errors.ErrNoRecord):
			app.helper.ClientError(w, http.StatusNotFound)
		default:
			app.helper.ServerError(w, err)
		}
		return
	}

	if err = app.helper.WriteJSON(w, http.StatusOK, helpers.Envelope{"worker": worker}, nil); err != nil {
		app.helper.ServerError(w, err)
		return
	}
}

func (app *application) getAllWorkers(w http.ResponseWriter, _ *http.Request) {
	workers, err := app.workerService.GetWorkers()
	if err != nil {
		switch {
		case errors.Is(err, custom_errors.ErrNoRecord):
			app.helper.ClientError(w, http.StatusNotFound)
		default:
			app.helper.ServerError(w, err)
		}
		return
	}

	if err = app.helper.WriteJSON(w, http.StatusOK, helpers.Envelope{"workers": workers}, nil); err != nil {
		app.helper.ServerError(w, err)
		return
	}
}
