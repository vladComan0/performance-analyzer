package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	"io"
	"net/http"
	"runtime/debug"
	"strconv"
)

type Helper struct {
	Log          zerolog.Logger
	DebugEnabled bool
}

func NewHelper(log zerolog.Logger, debugEnabled bool) *Helper {
	return &Helper{
		Log:          log,
		DebugEnabled: debugEnabled,
	}
}

type Envelope map[string]any

func (h *Helper) ClientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func (h *Helper) ServerError(w http.ResponseWriter, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	h.Log.Err(errors.New(trace)).Send()
	if h.DebugEnabled {
		http.Error(w, trace, http.StatusInternalServerError)
	}
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (h *Helper) ReadJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	const maxBytes = 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		// Custom Error Handling: Alex Edwards, Let's Go Further Chapter 4
		return err
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("body must only contain a single JSON object")
	}

	return nil
}

func (h *Helper) WriteJSON(w http.ResponseWriter, status int, data Envelope, headers http.Header) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		h.ServerError(w, err)
		return err
	}
	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, err = w.Write(js)
	if err != nil {
		return err
	}

	return nil
}

func (h *Helper) GetID(r *http.Request) (int, error) {
	// fetch the ID knowing that I use stdlib mux
	idString := r.PathValue("id")
	id, err := strconv.Atoi(idString)
	if err != nil {
		return 0, err
	}

	return id, nil
}
