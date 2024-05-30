package main

import (
	"errors"
	"fmt"
	"github.com/vladComan0/performance-analyzer/internal/models"
	"github.com/vladComan0/performance-analyzer/pkg/helpers"
	"net/http"
)

func (app *application) ping(w http.ResponseWriter, _ *http.Request) {
	_, err := w.Write([]byte("pong"))
	if err != nil {
		app.helper.ServerError(w, err)
		return
	}
}

func (app *application) createEnvironment(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name          string  `json:"name"`
		Endpoint      string  `json:"endpoint"`
		TokenEndpoint *string `json:"token_endpoint"`
		Username      *string `json:"username"`
		Password      *string `json:"password"`
		Disabled      *bool   `json:"disabled"`
	}

	if err := app.helper.ReadJSON(w, r, &input); err != nil {
		app.helper.ClientError(w, http.StatusBadRequest)
		return
	}

	var options []models.EnvironmentOption

	if input.TokenEndpoint != nil {
		options = append(options, models.WithEnvironmentTokenEndpoint(*input.TokenEndpoint))
	}

	if input.Username != nil {
		options = append(options, models.WithEnvironmentUsername(*input.Username))
	}

	if input.Password != nil {
		options = append(options, models.WithEnvironmentPassword(*input.Password))
	}

	if input.Disabled != nil {
		options = append(options, models.WithEnvironmentDisabled(*input.Disabled))
	}

	environment := models.NewEnvironment(input.Name, input.Endpoint, options...)

	id, err := app.environments.Insert(environment)
	if err != nil {
		app.helper.ServerError(w, err)
		return
	}

	environment, err = app.environments.Get(id)
	if err != nil {
		app.helper.ServerError(w, err)
		return
	}

	// Make the application aware of that new location -> add the headers to the right json helper function
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("v1/environments/%d", id))

	if err = app.helper.WriteJSON(w, http.StatusCreated, helpers.Envelope{"environment": environment}, headers); err != nil {
		app.helper.ServerError(w, err)
		return
	}

	app.infoLog.Printf("Created new environment with id: %d", id)
}

func (app *application) getEnvironment(w http.ResponseWriter, r *http.Request) {
	id, err := app.helper.GetID(r)
	if err != nil || id < 1 {
		app.helper.ClientError(w, http.StatusBadRequest)
		return
	}

	environment, err := app.environments.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrNoRecord):
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
	environments, err := app.environments.GetAll()
	if err != nil {
		switch {
		case errors.Is(err, models.ErrNoRecord):
			app.helper.ClientError(w, http.StatusNotFound)
		default:
			app.helper.ServerError(w, err)
		}
		return
	}

	if err := app.helper.WriteJSON(w, http.StatusOK, helpers.Envelope{"environments": environments}, nil); err != nil {
		app.helper.ServerError(w, err)
		return
	}
	app.infoLog.Printf("Retrieved all environments")
}

func (app *application) updateEnvironment(w http.ResponseWriter, r *http.Request) {
	id, err := app.helper.GetID(r)
	if err != nil || id < 1 {
		app.helper.ClientError(w, http.StatusBadRequest)
		return
	}

	environment, err := app.environments.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrNoRecord):
			app.helper.ClientError(w, http.StatusNotFound)
		default:
			app.helper.ServerError(w, err)
		}
		return
	}

	var input struct {
		Name          *string `json:"name"`
		Endpoint      *string `json:"endpoint"`
		TokenEndpoint *string `json:"token_endpoint"`
		Username      *string `json:"username"`
		Password      *string `json:"password"`
		Disabled      *bool   `json:"disabled"`
	}

	if err := app.helper.ReadJSON(w, r, &input); err != nil {
		app.helper.ClientError(w, http.StatusBadRequest)
		return
	}

	if input.Name != nil {
		environment.Name = *input.Name
	}

	if input.Endpoint != nil {
		environment.Endpoint = *input.Endpoint
	}

	if input.TokenEndpoint != nil {
		environment.TokenEndpoint = *input.TokenEndpoint
	}

	if input.Username != nil {
		environment.Username = *input.Username
	}

	if input.Password != nil {
		environment.Password = *input.Password
	}

	if input.Disabled != nil {
		environment.Disabled = *input.Disabled
	}

	err = app.environments.Update(environment)
	if err != nil {
		app.helper.ServerError(w, err)
		return
	}

	updatedEnvironment, err := app.environments.Get(environment.ID) // so that password is not returned
	if err != nil {
		app.helper.ServerError(w, err)
		return
	}

	if err = app.helper.WriteJSON(w, http.StatusOK, helpers.Envelope{"environment": updatedEnvironment}, nil); err != nil {
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

	if err = app.environments.Delete(id); err != nil {
		switch {
		case errors.Is(err, models.ErrNoRecord):
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

	app.infoLog.Printf("Deleted environment with id: %d", id)
}
