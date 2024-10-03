package entity

type Status string

const (
	StatusCreated  Status = "Created"
	StatusRunning  Status = "Running"
	StatusFinished Status = "Finished"
	StatusFailed   Status = "Failed"
)

func (w *Worker) SetStatus(s Status) {
	w.mu.Lock()
	defer w.mu.Unlock()
	switch s {
	case StatusCreated, StatusRunning, StatusFinished, StatusFailed:
		w.Status = s
	default:
		w.log.Error().Msgf("invalid status: %v", s)
	}
}

func (w *Worker) GetStatus() Status {
	return w.Status
}
