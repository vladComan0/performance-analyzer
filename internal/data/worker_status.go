package data

type Status string

const (
	StatusCreated  Status = "Created"
	StatusRunning  Status = "Running"
	StatusFinished Status = "Finished"
)

func (w *Worker) SetStatus(s Status) {
	switch s {
	case StatusCreated, StatusRunning, StatusFinished:
		w.Status = s
	default:

		w.errorLog.Printf("invalid status: %v", s)
	}
}

func (w *Worker) GetStatus() Status {
	return w.Status
}
